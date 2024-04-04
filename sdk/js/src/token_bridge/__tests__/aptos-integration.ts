import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { describe, expect, jest, test } from "@jest/globals";
import {
  AptosAccount,
  AptosClient,
  FaucetClient,
  HexString,
  Types,
} from "aptos";
import { ethers } from "ethers";
import { parseUnits } from "ethers/lib/utils";
import {
  APTOS_TOKEN_BRIDGE_EMITTER_ADDRESS,
  CHAIN_ID_APTOS,
  CHAIN_ID_ETH,
  CONTRACTS,
  approveEth,
  attestFromAptos,
  attestFromEth,
  createWrappedOnAptos,
  createWrappedOnEth,
  createWrappedTypeOnAptos,
  generateSignAndSubmitEntryFunction,
  generateSignAndSubmitScript,
  getEmitterAddressEth,
  getExternalAddressFromType,
  getForeignAssetAptos,
  getForeignAssetEth,
  getIsTransferCompletedAptos,
  getIsTransferCompletedEth,
  getIsWrappedAssetAptos,
  getOriginalAssetAptos,
  getSignedVAAWithRetry,
  hexToUint8Array,
  parseTokenTransferVaa,
  redeemOnAptos,
  redeemOnEth,
  transferFromAptos,
  transferFromEth,
  tryNativeToHexString,
  tryNativeToUint8Array,
  uint8ArrayToHex,
} from "../..";
import { registerCoin } from "../../aptos";
import {
  parseSequenceFromLogAptos,
  parseSequenceFromLogEth,
} from "../../bridge/parseSequenceFromLog";
import { TokenImplementation__factory } from "../../ethers-contracts";
import {
  APTOS_FAUCET_URL,
  APTOS_NODE_URL,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY6,
  TEST_ERC20,
  WORMHOLE_RPC_HOSTS,
} from "./utils/consts";

describe("Aptos SDK tests", () => {
  test("Transfer native token from Aptos to Ethereum", async () => {
    const APTOS_TOKEN_BRIDGE = CONTRACTS.DEVNET.aptos.token_bridge;
    const APTOS_CORE_BRIDGE = CONTRACTS.DEVNET.aptos.core;
    const COIN_TYPE = "0x1::aptos_coin::AptosCoin";

    // setup aptos
    const client = new AptosClient(APTOS_NODE_URL);
    const sender = new AptosAccount();
    const faucet = new FaucetClient(APTOS_NODE_URL, APTOS_FAUCET_URL);
    await faucet.fundAccount(sender.address(), 100_000_000);

    // attest native aptos token
    const attestPayload = attestFromAptos(
      APTOS_TOKEN_BRIDGE,
      CHAIN_ID_APTOS,
      COIN_TYPE
    );
    let tx = (await generateSignAndSubmitEntryFunction(
      client,
      sender,
      attestPayload
    )) as Types.UserTransaction;
    await client.waitForTransaction(tx.hash);

    // get signed attest vaa
    let sequence = parseSequenceFromLogAptos(APTOS_CORE_BRIDGE, tx);
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
    const provider = new ethers.providers.JsonRpcProvider(ETH_NODE_URL);
    const recipient = new ethers.Wallet(ETH_PRIVATE_KEY6, provider);
    const recipientAddress = await recipient.getAddress();
    const ethTokenBridge = CONTRACTS.DEVNET.ethereum.token_bridge;
    try {
      await createWrappedOnEth(ethTokenBridge, recipient, attestVAA);
    } catch (e) {
      // this could fail because the token is already attested (in an unclean env)
    }

    // check attestation on ethereum
    const externalAddress = hexToUint8Array(
      await getExternalAddressFromType(COIN_TYPE)
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
      await getBalanceAptos(client, COIN_TYPE, sender.address())
    );
    const transferPayload = transferFromAptos(
      APTOS_TOKEN_BRIDGE,
      COIN_TYPE,
      (10_000_000).toString(),
      CHAIN_ID_ETH,
      tryNativeToUint8Array(recipientAddress, CHAIN_ID_ETH)
    );
    tx = (await generateSignAndSubmitEntryFunction(
      client,
      sender,
      transferPayload
    )) as Types.UserTransaction;
    await client.waitForTransaction(tx.hash);
    const balanceAfterTransferAptos = ethers.BigNumber.from(
      await getBalanceAptos(client, COIN_TYPE, sender.address())
    );
    expect(
      balanceBeforeTransferAptos
        .sub(balanceAfterTransferAptos)
        .gt((10_000_000).toString())
    ).toBe(true);

    // get signed transfer vaa
    sequence = parseSequenceFromLogAptos(APTOS_CORE_BRIDGE, tx);
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
    const originAssetHex = tryNativeToUint8Array(COIN_TYPE, CHAIN_ID_APTOS);
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
  });
  test("Transfer native ERC-20 from Ethereum to Aptos", async () => {
    // setup ethereum
    const provider = new ethers.providers.JsonRpcProvider(ETH_NODE_URL);
    const sender = new ethers.Wallet(ETH_PRIVATE_KEY6, provider);
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

    await provider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`
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
    const recipient = new AptosAccount();
    const faucet = new FaucetClient(APTOS_NODE_URL, APTOS_FAUCET_URL);
    await faucet.fundAccount(recipient.address(), 100_000_000);
    const aptosTokenBridge = CONTRACTS.DEVNET.aptos.token_bridge;
    const createWrappedCoinTypePayload = createWrappedTypeOnAptos(
      aptosTokenBridge,
      attestVAA
    );
    try {
      const tx = await generateSignAndSubmitEntryFunction(
        client,
        recipient,
        createWrappedCoinTypePayload
      );
      await client.waitForTransaction(tx.hash);
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
      const tx = await generateSignAndSubmitEntryFunction(
        client,
        recipient,
        createWrappedCoinPayload
      );
      await client.waitForTransaction(tx.hash);
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
    const aptosWrappedType = await getForeignAssetAptos(
      client,
      aptosTokenBridge,
      CHAIN_ID_ETH,
      TEST_ERC20
    );
    if (!aptosWrappedType) {
      throw new Error("Failed to create wrapped coin on Aptos");
    }

    const info = await getOriginalAssetAptos(
      client,
      aptosTokenBridge,
      aptosWrappedType
    );
    expect(uint8ArrayToHex(info.assetAddress)).toEqual(
      tryNativeToHexString(TEST_ERC20, CHAIN_ID_ETH)
    );
    expect(info.chainId).toEqual(CHAIN_ID_ETH);
    expect(info.isWrapped).toEqual(
      await getIsWrappedAssetAptos(client, aptosTokenBridge, aptosWrappedType)
    );

    // transfer from eth
    const balanceBeforeTransferEth = await getBalanceEth(TEST_ERC20, sender);
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

    await provider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`
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

    // register token on aptos
    const script = registerCoin(aptosTokenBridge, CHAIN_ID_ETH, TEST_ERC20);
    await generateSignAndSubmitScript(client, recipient, script);

    // redeem on aptos
    const balanceBeforeTransferAptos = ethers.BigNumber.from(
      await getBalanceAptos(client, aptosWrappedType, recipient.address())
    );
    const redeemPayload = await redeemOnAptos(
      client,
      aptosTokenBridge,
      transferVAA
    );
    const tx = await generateSignAndSubmitEntryFunction(
      client,
      recipient,
      redeemPayload
    );
    await client.waitForTransaction(tx.hash);
    expect(
      await getIsTransferCompletedAptos(client, aptosTokenBridge, transferVAA)
    ).toBe(true);

    // check balances
    const balanceAfterTransferAptos = ethers.BigNumber.from(
      await getBalanceAptos(client, aptosWrappedType, recipient.address())
    );
    expect(
      balanceAfterTransferAptos.sub(balanceBeforeTransferAptos).toString()
    ).toEqual(parseUnits("1", 8).toString()); // max decimals is 8
    const balanceAfterTransferEth = await getBalanceEth(TEST_ERC20, sender);
    expect(
      balanceBeforeTransferEth.sub(balanceAfterTransferEth).toString()
    ).toEqual(amount.toString());
  });
  test("Transfer native token with payload from Aptos to Ethereum", async () => {
    const APTOS_TOKEN_BRIDGE = CONTRACTS.DEVNET.aptos.token_bridge;
    const APTOS_CORE_BRIDGE = CONTRACTS.DEVNET.aptos.core;
    const COIN_TYPE = "0x1::aptos_coin::AptosCoin";

    // setup aptos
    const client = new AptosClient(APTOS_NODE_URL);
    const sender = new AptosAccount();
    const faucet = new FaucetClient(APTOS_NODE_URL, APTOS_FAUCET_URL);
    await faucet.fundAccount(sender.address(), 100_000_000);

    // attest native aptos token
    const attestPayload = attestFromAptos(
      APTOS_TOKEN_BRIDGE,
      CHAIN_ID_APTOS,
      COIN_TYPE
    );
    let tx = (await generateSignAndSubmitEntryFunction(
      client,
      sender,
      attestPayload
    )) as Types.UserTransaction;
    await client.waitForTransaction(tx.hash);

    // get signed attest vaa
    let sequence = parseSequenceFromLogAptos(APTOS_CORE_BRIDGE, tx);
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
    const provider = new ethers.providers.JsonRpcProvider(ETH_NODE_URL);
    const recipient = new ethers.Wallet(ETH_PRIVATE_KEY6, provider);
    const recipientAddress = await recipient.getAddress();
    const ethTokenBridge = CONTRACTS.DEVNET.ethereum.token_bridge;
    try {
      await createWrappedOnEth(ethTokenBridge, recipient, attestVAA);
    } catch (e) {
      // this could fail because the token is already attested (in an unclean env)
    }

    // check attestation on ethereum
    const externalAddress = hexToUint8Array(
      await getExternalAddressFromType(COIN_TYPE)
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
      await getBalanceAptos(client, COIN_TYPE, sender.address())
    );
    const payload = Buffer.from("All your base are belong to us");
    const transferPayload = transferFromAptos(
      APTOS_TOKEN_BRIDGE,
      COIN_TYPE,
      (10_000_000).toString(),
      CHAIN_ID_ETH,
      tryNativeToUint8Array(recipientAddress, CHAIN_ID_ETH),
      "0",
      payload
    );
    tx = (await generateSignAndSubmitEntryFunction(
      client,
      sender,
      transferPayload
    )) as Types.UserTransaction;
    await client.waitForTransaction(tx.hash);
    const balanceAfterTransferAptos = ethers.BigNumber.from(
      await getBalanceAptos(client, COIN_TYPE, sender.address())
    );
    expect(
      balanceBeforeTransferAptos
        .sub(balanceAfterTransferAptos)
        .gt((10_000_000).toString())
    ).toBe(true);

    // get signed transfer vaa
    sequence = parseSequenceFromLogAptos(APTOS_CORE_BRIDGE, tx);
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
    const { tokenTransferPayload } = parseTokenTransferVaa(transferVAA);
    expect(tokenTransferPayload.toString()).toBe(payload.toString());

    // get balance on eth
    const originAssetHex = tryNativeToUint8Array(COIN_TYPE, CHAIN_ID_APTOS);
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
