import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { afterAll, beforeAll, describe, expect, test } from "@jest/globals";
import {
  Connection,
  Ed25519Keypair,
  JsonRpcProvider,
  RawSigner,
  fromB64,
  getMoveObjectType,
  getPublishedObjectChanges,
} from "@mysten/sui.js";
import { ethers } from "ethers";
import {
  attestFromEth,
  attestFromSui,
  createWrappedOnSui,
  createWrappedOnSuiPrepare,
  getEmitterAddressEth,
  getIsWrappedAssetSui,
  getSignedVAAWithRetry,
  parseSequenceFromLogEth,
} from "../..";
import {
  executeTransactionBlock,
  getInnerType,
  getWrappedCoinType,
} from "../../sui";
import { CHAIN_ID_ETH, CONTRACTS, SUI_OBJECT_IDS } from "../../utils";
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY10,
  SUI_FAUCET_URL,
  SUI_NODE_URL,
  TEST_ERC20,
  WORMHOLE_RPC_HOSTS,
} from "./utils/consts";
import { assertIsNotNullOrUndefined } from "./utils/helpers";

const JEST_TEST_TIMEOUT = 60000;

// Sui constants
const SUI_CORE_BRIDGE_ADDRESS = CONTRACTS.DEVNET.sui.core;
const SUI_TOKEN_BRIDGE_ADDRESS = CONTRACTS.DEVNET.sui.token_bridge;
const SUI_CORE_BRIDGE_STATE_OBJECT_ID = SUI_OBJECT_IDS.DEVNET.core_state;
const SUI_TOKEN_BRIDGE_STATE_OBJECT_ID =
  SUI_OBJECT_IDS.DEVNET.token_bridge_state;
const SUI_DEPLOYER_PRIVATE_KEY = "AGA20wtGcwbcNAG4nwapbQ5wIuXwkYQEWFUoSVAxctHb";

const suiKeypair: Ed25519Keypair = Ed25519Keypair.fromSecretKey(
  fromB64(SUI_DEPLOYER_PRIVATE_KEY).slice(1)
);
const suiAddress: string = suiKeypair.getPublicKey().toSuiAddress();
const suiProvider: JsonRpcProvider = new JsonRpcProvider(
  new Connection({
    fullnode: SUI_NODE_URL,
    faucet: SUI_FAUCET_URL,
  })
);
const suiSigner: RawSigner = new RawSigner(suiKeypair, suiProvider);

// Eth constants
const ETH_CORE_BRIDGE_ADDRESS = CONTRACTS.DEVNET.ethereum.core;
const ETH_TOKEN_BRIDGE_ADDRESS = CONTRACTS.DEVNET.ethereum.token_bridge;

const ethProvider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
const ethSigner = new ethers.Wallet(ETH_PRIVATE_KEY10, ethProvider);

beforeAll(async () => {
  expect(SUI_CORE_BRIDGE_ADDRESS).toBeDefined();
  expect(SUI_TOKEN_BRIDGE_ADDRESS).toBeDefined();
  expect(SUI_CORE_BRIDGE_STATE_OBJECT_ID).toBeDefined();
  expect(SUI_TOKEN_BRIDGE_STATE_OBJECT_ID).toBeDefined();
});

afterAll(async () => {
  await ethProvider.destroy();
});

describe("Sui SDK tests", () => {
  test("Transfer native ERC-20 from Ethereum to Sui", async () => {
    // Attest on Eth
    const ethAttestTxRes = await attestFromEth(
      ETH_TOKEN_BRIDGE_ADDRESS,
      ethSigner,
      TEST_ERC20
    );

    // Get attest VAA
    const sequence = parseSequenceFromLogEth(
      ethAttestTxRes,
      ETH_CORE_BRIDGE_ADDRESS
    );
    expect(sequence).toBeTruthy();

    const { vaaBytes: attestVAA } = await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      CHAIN_ID_ETH,
      getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS),
      sequence,
      {
        transport: NodeHttpTransport(),
      },
      1000,
      5
    );
    expect(attestVAA).toBeTruthy();

    // Start create wrapped on Sui
    // const MOCK_VAA =
    //   "0100000000010026ff86c07ef853ef955a63c58a8d08eeb2ac232b91e725bd41baeb3c05c5c18d07aef3c02dc3d5ca8ad0600a447c3d55386d0a0e85b23378d438fbb1e207c3b600000002c3a86f000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000001020000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a000212544b4e0000000000000000000000000000000000000000000000000000000000457468657265756d205465737420546f6b656e00000000000000000000000000";
    const suiPrepareRegistrationTxPayload = await createWrappedOnSuiPrepare(
      "DEVNET",
      SUI_CORE_BRIDGE_ADDRESS,
      SUI_TOKEN_BRIDGE_ADDRESS,
      attestVAA,
      suiAddress
    );
    const suiPrepareRegistrationTxRes = await executeTransactionBlock(
      suiSigner,
      suiPrepareRegistrationTxPayload
    );
    expect(suiPrepareRegistrationTxRes.effects?.status.status).toBe("success");

    // Complete create wrapped on Sui
    const publishEvents = getPublishedObjectChanges(
      suiPrepareRegistrationTxRes
    );
    expect(publishEvents.length).toBe(1);

    const coinPackageId = publishEvents[0].packageId;
    const suiCompleteRegistrationTxPayload = await createWrappedOnSui(
      suiProvider,
      SUI_TOKEN_BRIDGE_ADDRESS,
      SUI_CORE_BRIDGE_STATE_OBJECT_ID,
      SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
      suiAddress,
      coinPackageId
    );
    const suiCompleteRegistrationTxRes = await executeTransactionBlock(
      suiSigner,
      suiCompleteRegistrationTxPayload
    );
    expect(suiCompleteRegistrationTxRes.effects?.status.status).toBe("success");
    expect(
      await getIsWrappedAssetSui(
        suiProvider,
        SUI_TOKEN_BRIDGE_ADDRESS,
        SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
        getWrappedCoinType(coinPackageId)
      )
    ).toBe(true);
  });
  test("Transfer non-SUI Sui token to Ethereum", async () => {
    // Get COIN_8 coin type
    const res = await suiProvider.getOwnedObjects({
      owner: suiAddress,
      options: { showContent: true, showType: true },
    });
    const coins = res.data.filter((o) =>
      (o.data?.type ?? "").includes("COIN_8")
    );
    expect(coins.length).toBe(1);

    const coin8 = coins[0];
    const coin8Type = getInnerType(getMoveObjectType(coin8) ?? "");
    const coin8TreasuryCapObjectId = coin8.data?.objectId;
    assertIsNotNullOrUndefined(coin8Type);
    assertIsNotNullOrUndefined(coin8TreasuryCapObjectId);
    expect(
      await getIsWrappedAssetSui(
        suiProvider,
        SUI_TOKEN_BRIDGE_ADDRESS,
        SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
        coin8Type
      )
    ).toBe(false);

    // Attest on Sui
    const suiAttestTxPayload = await attestFromSui(
      suiProvider,
      SUI_TOKEN_BRIDGE_ADDRESS,
      SUI_CORE_BRIDGE_STATE_OBJECT_ID,
      SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
      coin8Type,
      0
    );
    const suiAttestTxRes = await executeTransactionBlock(
      suiSigner,
      suiAttestTxPayload
    );
    expect(suiAttestTxRes.effects?.status.status).toBe("success");

    // transfer tokens to Ethereum
    // const coinsObject = (
    //   await suiProvider.getGasObjectsOwnedByAddress(suiAddress)
    // )[1];
    // const suiTransferTxPayload = await transferFromSui(
    //   suiProvider,
    //   SUI_CORE_BRIDGE_ADDRESS,
    //   SUI_TOKEN_BRIDGE_ADDRESS,
    //   SUI_COIN_TYPE,
    //   coinsObject.objectId,
    //   feeObject.objectId,
    //   CHAIN_ID_ETH,
    //   tryNativeToUint8Array(ethSigner.address, CHAIN_ID_ETH)
    // );
    // const suiTransferTxResult = await executeTransaction(
    //   suiSigner,
    //   suiTransferTxPayload
    // );
    // console.log(
    //   "suiTransferTxResult",
    //   JSON.stringify(suiTransferTxResult, null, 2)
    // );

    // fetch vaa

    // redeem on Ethereum

    // transfer tokens back to Sui
  });
});
