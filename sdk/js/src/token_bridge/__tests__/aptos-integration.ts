import { describe, expect, jest, test } from "@jest/globals";
import { AptosAccount, AptosClient, FaucetClient, HexString } from "aptos";
import {
  approveEth,
  APTOS_TOKEN_BRIDGE_EMITTER_ADDRESS,
  attestFromAptos,
  attestFromEth,
  CHAIN_ID_APTOS,
  CHAIN_ID_ETH,
  CONTRACTS,
  createWrappedOnAptos,
  createWrappedOnEth,
  createWrappedTypeOnAptos,
  getAssetFullyQualifiedType,
  getEmitterAddressEth,
  getExternalAddressFromType,
  getForeignAssetAptos,
  getForeignAssetEth,
  getIsTransferCompletedAptos,
  getIsTransferCompletedEth,
  getSignedVAAWithRetry,
  hexToUint8Array,
  redeemOnAptos,
  redeemOnEth,
  TokenImplementation__factory,
  transferFromAptos,
  transferFromEth,
  tryNativeToUint8Array,
  uint8ArrayToHex,
  waitForSignAndSubmitTransaction,
} from "../..";
import { setDefaultWasm } from "../../solana/wasm";
import {
  APTOS_FAUCET_URL,
  APTOS_NODE_URL,
  APTOS_PRIVATE_KEY,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY3,
  TEST_ERC20,
  WORMHOLE_RPC_HOSTS,
} from "./consts";
import {
  parseSequenceFromLogAptos,
  parseSequenceFromLogEth,
} from "../../bridge/parseSequenceFromLog";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { ethers } from "ethers";
import { parseUnits } from "ethers/lib/utils";

setDefaultWasm("node");

const JEST_TEST_TIMEOUT = 60000;
jest.setTimeout(JEST_TEST_TIMEOUT);

describe("Aptos SDK tests", () => {
  test("Transfer native token from Aptos to Ethereum", async () => {
    // setup aptos
    const client = new AptosClient(APTOS_NODE_URL);
    const faucet = new FaucetClient(APTOS_NODE_URL, APTOS_FAUCET_URL);
    const sender = new AptosAccount(hexToUint8Array(APTOS_PRIVATE_KEY));
    const aptosTokenBridge = CONTRACTS.DEVNET.aptos.token_bridge;
    const aptosCoreBridge = CONTRACTS.DEVNET.aptos.core;

    // sanity check funds in the account
    const coinType = "0x1::aptos_coin::AptosCoin";
    const before = await getBalanceAptos(client, coinType, sender.address());
    await faucet.fundAccount(sender.address(), 100_000_000);
    const after = await getBalanceAptos(client, coinType, sender.address());
    expect(Number(after) - Number(before)).toEqual(100_000_000);

    // attest native aptos token
    const attestPayload = attestFromAptos(
      aptosTokenBridge,
      CHAIN_ID_APTOS,
      coinType
    );
    let tx = await waitForSignAndSubmitTransaction(
      client,
      sender,
      attestPayload
    );

    // get signed attest vaa
    let sequence = parseSequenceFromLogAptos(aptosCoreBridge, tx);
    expect(sequence).toBeTruthy();

    const { vaaBytes: attestVAA } = await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      CHAIN_ID_APTOS,
      APTOS_TOKEN_BRIDGE_EMITTER_ADDRESS,
      sequence!,
      {
        transport: NodeHttpTransport(),
      }
    );
    expect(attestVAA).toBeTruthy();

    // setup ethereum
    const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
    const recipient = new ethers.Wallet(ETH_PRIVATE_KEY3, provider);
    const recipientAddress = await recipient.getAddress();
    const ethTokenBridge = CONTRACTS.DEVNET.ethereum.token_bridge;
    try {
      await createWrappedOnEth(ethTokenBridge, recipient, attestVAA);
    } catch (e) {
      // this could fail because the token is already attested (in an unclean env)
    }

    // check attestation on ethereum
    const externalAddress = hexToUint8Array(
      await getExternalAddressFromType(coinType)
    );
    const address = getForeignAssetEth(
      ethTokenBridge,
      provider,
      CHAIN_ID_APTOS,
      externalAddress
    );
    expect(address).toBeTruthy();
    expect(address).not.toBe(ethers.constants.AddressZero);

    // transfer from aptos
    const balanceBeforeTransferAptos = ethers.BigNumber.from(
      await getBalanceAptos(client, coinType, sender.address())
    );
    const transferPayload = transferFromAptos(
      aptosTokenBridge,
      coinType,
      (10_000_000).toString(),
      CHAIN_ID_ETH,
      tryNativeToUint8Array(recipientAddress, CHAIN_ID_ETH)
    );
    tx = await waitForSignAndSubmitTransaction(client, sender, transferPayload);
    const balanceAfterTransferAptos = ethers.BigNumber.from(
      await getBalanceAptos(client, coinType, sender.address())
    );
    expect(
      balanceBeforeTransferAptos
        .sub(balanceAfterTransferAptos)
        .gt((10_000_000).toString())
    ).toBe(true);

    // get signed transfer vaa
    sequence = parseSequenceFromLogAptos(aptosCoreBridge, tx);
    expect(sequence).toBeTruthy();

    const { vaaBytes: transferVAA } = await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      CHAIN_ID_APTOS,
      APTOS_TOKEN_BRIDGE_EMITTER_ADDRESS,
      sequence!,
      {
        transport: NodeHttpTransport(),
      }
    );
    expect(transferVAA).toBeTruthy();

    // get balance on eth
    const originAssetHex = tryNativeToUint8Array(coinType, CHAIN_ID_APTOS);
    if (!originAssetHex) {
      throw new Error("originAssetHex is null");
    }

    const foreignAsset = await getForeignAssetEth(
      ethTokenBridge,
      provider,
      CHAIN_ID_APTOS,
      originAssetHex
    );
    if (!foreignAsset) {
      throw new Error("foreignAsset is null");
    }

    const balanceBeforeTransferEth = await getBalanceEth(
      foreignAsset,
      recipient
    );

    // redeem on eth
    await redeemOnEth(ethTokenBridge, recipient, transferVAA);
    expect(
      await getIsTransferCompletedEth(ethTokenBridge, provider, transferVAA)
    ).toBe(true);
    const balanceAfterTransferEth = await getBalanceEth(
      foreignAsset,
      recipient
    );
    expect(
      balanceAfterTransferEth.sub(balanceBeforeTransferEth).toNumber()
    ).toEqual(10_000_000);

    // clean up
    provider.destroy();
  });
  test("Transfer native ERC-20 from Ethereum to Aptos", async () => {
    // setup ethereum
    const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
    const sender = new ethers.Wallet(ETH_PRIVATE_KEY3, provider);
    const senderAddress = await sender.getAddress();
    const ethTokenBridge = CONTRACTS.DEVNET.ethereum.token_bridge;
    const ethCoreBridge = CONTRACTS.DEVNET.ethereum.core;

    // attest from eth
    const attestReceipt = await attestFromEth(
      ethTokenBridge,
      sender,
      TEST_ERC20
    );

    // get signed attest vaa
    let sequence = parseSequenceFromLogEth(attestReceipt, ethCoreBridge);
    expect(sequence).toBeTruthy();

    const { vaaBytes: attestVAA } = await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      CHAIN_ID_ETH,
      getEmitterAddressEth(ethTokenBridge),
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );
    expect(attestVAA).toBeTruthy();

    // setup aptos
    const client = new AptosClient(APTOS_NODE_URL);
    const recipient = new AptosAccount(hexToUint8Array(APTOS_PRIVATE_KEY));
    const aptosTokenBridge = CONTRACTS.DEVNET.aptos.token_bridge;
    const createWrappedCoinTypePayload = createWrappedTypeOnAptos(
      aptosTokenBridge,
      attestVAA
    );
    try {
      await waitForSignAndSubmitTransaction(
        client,
        recipient,
        createWrappedCoinTypePayload
      );
    } catch (e) {
      // only throw if token has not been attested but this call fails
      if (
        !(
          new Error(e).message.includes("ECOIN_INFO_ALREADY_PUBLISHED") ||
          new Error(e).message.includes("ERESOURCE_ACCCOUNT_EXISTS")
        )
      ) {
        throw e;
      }
    }

    const createWrappedCoinPayload = createWrappedOnAptos(
      aptosTokenBridge,
      attestVAA
    );
    try {
      await waitForSignAndSubmitTransaction(
        client,
        recipient,
        createWrappedCoinPayload
      );
    } catch (e) {
      // only throw if token has not been attested but this call fails
      if (
        !(
          new Error(e).message.includes("ECOIN_INFO_ALREADY_PUBLISHED") ||
          new Error(e).message.includes("ERESOURCE_ACCCOUNT_EXISTS")
        )
      ) {
        throw e;
      }
    }

    // check attestation on aptos
    const aptosWrappedAddress = await getForeignAssetAptos(
      client,
      aptosTokenBridge,
      CHAIN_ID_ETH,
      TEST_ERC20
    );
    if (!aptosWrappedAddress) {
      throw new Error("Failed to create wrapped coin on Aptos");
    }

    // get balances
    const wrappedType = getAssetFullyQualifiedType(
      aptosTokenBridge,
      CHAIN_ID_ETH,
      TEST_ERC20
    );
    if (!wrappedType) {
      throw new Error("wrappedType is null");
    }

    const balanceBeforeTransferAptos = ethers.BigNumber.from(
      await getBalanceAptos(client, wrappedType, recipient.address())
    );
    const balanceBeforeTransferEth = await getBalanceEth(TEST_ERC20, sender);

    // transfer from eth
    const amount = parseUnits("1", 18);
    await approveEth(ethTokenBridge, TEST_ERC20, sender, amount);
    const transferReceipt = await transferFromEth(
      ethTokenBridge,
      sender,
      TEST_ERC20,
      amount,
      CHAIN_ID_APTOS,
      tryNativeToUint8Array(recipient.address().hex(), CHAIN_ID_APTOS)
    );

    // get signed transfer vaa
    sequence = parseSequenceFromLogEth(transferReceipt, ethCoreBridge);
    expect(sequence).toBeTruthy();

    const { vaaBytes: transferVAA } = await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      CHAIN_ID_ETH,
      getEmitterAddressEth(ethTokenBridge),
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );
    expect(transferVAA).toBeTruthy();

    // redeem on aptos
    const redeemPayload = await redeemOnAptos(
      client,
      aptosTokenBridge,
      transferVAA
    );
    await waitForSignAndSubmitTransaction(client, recipient, redeemPayload);
    expect(
      await getIsTransferCompletedAptos(client, aptosTokenBridge, transferVAA)
    ).toBe(true);

    // check balances
    const balanceAfterTransferAptos = ethers.BigNumber.from(
      await getBalanceAptos(client, wrappedType, recipient.address())
    );
    expect(
      balanceAfterTransferAptos.sub(balanceBeforeTransferAptos).toString()
    ).toEqual(parseUnits("1", 8).toString()); // max decimals is 8
    const balanceAfterTransferEth = await getBalanceEth(TEST_ERC20, sender);
    expect(
      balanceBeforeTransferEth.sub(balanceAfterTransferEth).toString()
    ).toEqual(amount.toString());
  });
});

const getBalanceAptos = async (
  client: AptosClient,
  type: string,
  address: HexString
): Promise<string> => {
  const res = await client.getAccountResource(
    address,
    `0x1::coin::CoinStore<${type}>`
  );
  return (res.data as any).coin.value;
};

const getBalanceEth = (tokenAddress: string, wallet: ethers.Wallet) => {
  let token = TokenImplementation__factory.connect(tokenAddress, wallet);
  return token.balanceOf(wallet.address);
};
