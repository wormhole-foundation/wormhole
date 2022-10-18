import { describe, expect, jest, test } from "@jest/globals";
import { AptosAccount, AptosClient, FaucetClient, HexString } from "aptos";
import {
  APTOS_TOKEN_BRIDGE_EMITTER_ADDRESS,
  attestFromAptos,
  CHAIN_ID_APTOS,
  CHAIN_ID_ETH,
  CONTRACTS,
  createWrappedOnEth,
  getExternalAddressFromType,
  getForeignAssetEth,
  getIsTransferCompletedEth,
  getSignedVAAWithRetry,
  hexToUint8Array,
  redeemOnEth,
  TokenImplementation__factory,
  transferFromAptos,
  tryNativeToHexString,
  tryNativeToUint8Array,
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
import { parseSequenceFromLogAptos } from "../../bridge/parseSequenceFromLog";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { ethers } from "ethers";
import {
  createWrappedCoin,
  createWrappedCoinType,
  registerChain,
} from "../../aptos";

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
    const before = await getBalanceAptos(client, sender.address());
    await faucet.fundAccount(sender.address(), 100_000_000);
    const after = await getBalanceAptos(client, sender.address());
    expect(after - before).toEqual(100_000_000);

    // attest native aptos token
    const coinType = "0x1::aptos_coin::AptosCoin";
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
    console.log(await getExternalAddressFromType(coinType));
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

    // transfer to ethereum
    const balanceBeforeTransferAptos = await getBalanceAptos(
      client,
      sender.address()
    );
    const transferPayload = transferFromAptos(
      aptosTokenBridge,
      coinType,
      (10_000_000).toString(),
      CHAIN_ID_ETH,
      tryNativeToUint8Array(recipientAddress, CHAIN_ID_ETH)
    );
    tx = await waitForSignAndSubmitTransaction(client, sender, transferPayload);
    const balanceAfterTransferAptos = await getBalanceAptos(
      client,
      sender.address()
    );
    expect(
      balanceBeforeTransferAptos - balanceAfterTransferAptos
    ).toBeGreaterThan(10_000_000);

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
    const originAssetHex = tryNativeToHexString(coinType, CHAIN_ID_APTOS);
    if (!originAssetHex) {
      throw new Error("originAssetHex is null");
    }

    const foreignAsset = await getForeignAssetEth(
      ethTokenBridge,
      provider,
      CHAIN_ID_APTOS,
      hexToUint8Array(originAssetHex)
    );
    if (!foreignAsset) {
      throw new Error("foreignAsset is null");
    }

    let token = TokenImplementation__factory.connect(foreignAsset, recipient);
    const balanceBeforeTransferEth = await token.balanceOf(recipientAddress);

    // redeem on eth
    await redeemOnEth(ethTokenBridge, recipient, transferVAA, {
      gasLimit: 5000000,
    });
    expect(
      await getIsTransferCompletedEth(ethTokenBridge, provider, transferVAA)
    ).toBe(true);
    const balanceAfterTransferEth = await token.balanceOf(recipientAddress);
    expect(
      balanceAfterTransferEth.sub(balanceBeforeTransferEth).toNumber()
    ).toEqual(10_000_000);

    // clean up
    provider.destroy();
  });
  test("Transfer from Ethereum to Aptos", async () => {
    // setup aptos
    const client = new AptosClient(APTOS_NODE_URL);
    const account = new AptosAccount(hexToUint8Array(APTOS_PRIVATE_KEY));

    // // setup ethereum
    // const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
    // const signer = new ethers.Wallet(ETH_PRIVATE_KEY3, provider);
    // try {
    //   // await attestFromEth(CONTRACTS.DEVNET.ethereum.token_bridge, signer, TEST_ERC20);
    // } catch (e) {
    //   // this could fail because the token is already attested (in an unclean env)
    // }

    // register eth
    const tokenBridgeAddress = CONTRACTS.DEVNET.aptos.token_bridge;
    const registrationVAA = hexToUint8Array(
      "010000000001002cef01cdddcbf42adeb88eb10ddb6bfb8dd7beb97bd61548e34b299dc47c80fd6b5a6679c345aef1b868c913d46d8d33a1849bb7c11bbcb791a2cee9caac8aa4000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000001c53fe200000000000000000000000000000000000000000000546f6b656e42726964676501000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
    );
    const registrationPayload = registerChain(
      tokenBridgeAddress,
      registrationVAA
    );
    try {
      await waitForSignAndSubmitTransaction(
        client,
        account,
        registrationPayload
      );
    } catch {} // hack to run test multiple times

    // get attest vaa
    const attestVAA = hexToUint8Array(
      "010000000001008b63d1b99fe9052678bc8be230c2a0b6d911368a9c5e4c5e1f3251de91342c3d04a18236bf7f53383893a98a2b74e6c8c7afd63e68b117766e14a25ac26e0be301000000010000000100020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c1600000000028b11e300020000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002085445535400000000000000000000000000000000000000000000000000000000544553545f455243323000000000000000000000000000000000000000000000"
    );

    // create cointype
    const createWrappedCoinTypePayload = createWrappedCoinType(
      tokenBridgeAddress,
      attestVAA
    );
    try {
      await waitForSignAndSubmitTransaction(
        client,
        account,
        createWrappedCoinTypePayload
      );
    } catch {} // hack to run test multiple times

    // create coin
    const createWrappedCoinPayload = createWrappedCoin(
      tokenBridgeAddress,
      attestVAA
    );
    await waitForSignAndSubmitTransaction(
      client,
      account,
      createWrappedCoinPayload
    );
  });
});

const getBalanceAptos = async (
  client: AptosClient,
  address: HexString
): Promise<number> => {
  const type = "0x1::coin::CoinStore<0x1::aptos_coin::AptosCoin>";
  const res = await client.getAccountResource(address, type);
  return (res.data as any).coin.value;
};
