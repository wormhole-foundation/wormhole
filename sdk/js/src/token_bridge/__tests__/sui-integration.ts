import {
  afterAll,
  beforeAll,
  beforeEach,
  describe,
  expect,
  jest,
  test,
} from "@jest/globals";
import {
  Connection,
  Ed25519Keypair,
  JsonRpcProvider,
  RawSigner,
  fromB64,
  getPublishedObjectChanges,
} from "@mysten/sui.js";
import { executeTransactionBlock } from "../../sui";
import { CONTRACTS, SUI_OBJECT_IDS } from "../../utils";
import {
  createWrappedOnSui,
  createWrappedOnSuiPrepare,
} from "../createWrapped";
import { SUI_FAUCET_URL, SUI_NODE_URL } from "./utils/consts";

const JEST_TEST_TIMEOUT = 60000;
jest.setTimeout(JEST_TEST_TIMEOUT);

const SUI_CORE_BRIDGE_ADDRESS = CONTRACTS.DEVNET.sui.core;
const SUI_TOKEN_BRIDGE_ADDRESS = CONTRACTS.DEVNET.sui.token_bridge;
const SUI_CORE_BRIDGE_STATE_OBJECT_ID = SUI_OBJECT_IDS.DEVNET.core_state;
const SUI_TOKEN_BRIDGE_STATE_OBJECT_ID =
  SUI_OBJECT_IDS.DEVNET.token_bridge_state;
const SUI_DEPLOYER_PRIVATE_KEY = "AGA20wtGcwbcNAG4nwapbQ5wIuXwkYQEWFUoSVAxctHb";

// const ethProvider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
// const ethSigner = new ethers.Wallet(ETH_PRIVATE_KEY, ethProvider);

let suiKeypair: Ed25519Keypair;
let suiAddress: string;
let suiProvider: JsonRpcProvider;
let suiSigner: RawSigner;

beforeAll(async () => {
  expect(SUI_CORE_BRIDGE_ADDRESS).toBeDefined();
  expect(SUI_TOKEN_BRIDGE_ADDRESS).toBeDefined();
  expect(SUI_CORE_BRIDGE_STATE_OBJECT_ID).toBeDefined();
  expect(SUI_TOKEN_BRIDGE_STATE_OBJECT_ID).toBeDefined();
});

beforeEach(async () => {
  suiKeypair = Ed25519Keypair.fromSecretKey(
    fromB64(SUI_DEPLOYER_PRIVATE_KEY).slice(1)
  );
  suiAddress = suiKeypair.getPublicKey().toSuiAddress();
  suiProvider = new JsonRpcProvider(
    new Connection({
      fullnode: SUI_NODE_URL,
      faucet: SUI_FAUCET_URL,
    })
  );
  suiSigner = new RawSigner(suiKeypair, suiProvider);
});

afterAll(async () => {
  // await ethProvider.destroy();
});

describe("Sui SDK tests", () => {
  test("Transfer native Sui token to Ethereum and back", async () => {
    // // Get COIN_8 coin type
    // const res = await suiProvider.getOwnedObjects({
    //   owner: suiAddress,
    //   options: { showContent: true, showType: true },
    // });
    // console.log(JSON.stringify(res.data, null, 2));
    // const coins = res.data.filter((o) =>
    //   (o.data?.type ?? "").includes("COIN_8")
    // );
    // expect(coins.length).toBe(1);
    // const coin8 = coins[0];
    // const coin8Type = getInnerType(getMoveObjectType(coin8) ?? "");
    // const coin8TreasuryCapObjectId = coin8.data?.objectId;
    // assertIsNotNullOrUndefined(coin8Type);
    // assertIsNotNullOrUndefined(coin8TreasuryCapObjectId);

    // Attest on Sui
    // const suiAttestTxPayload = await attestFromSui(
    //   suiProvider,
    //   SUI_TOKEN_BRIDGE_ADDRESS,
    //   SUI_CORE_STATE_OBJECT_ID,
    //   SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
    //   coin8Type,
    //   0
    // );
    // const suiAttestTxRes = await executeTransactionBlock(
    //   suiSigner,
    //   suiAttestTxPayload
    // );
    // expect(suiAttestTxRes.effects?.status.status).toBe("success");

    // Start create wrapped on Sui
    const MOCK_VAA =
      "0100000000010080366065746148420220f25a6275097370e8db40984529a6676b7a5fc9feb11755ec49ca626b858ddfde88d15601f85ab7683c5f161413b0412143241c700aff010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef000000000150eb23000200000000000000000000000000000000000000000000000000000000beefface00020c424545460000000000000000000000000000000000000000000000000000000042656566206661636520546f6b656e0000000000000000000000000000000000";
    const suiPrepareRegistrationTxPayload = await createWrappedOnSuiPrepare(
      "DEVNET",
      SUI_CORE_BRIDGE_ADDRESS,
      SUI_TOKEN_BRIDGE_ADDRESS,
      Buffer.from(MOCK_VAA, "hex"),
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
    console.log(JSON.stringify(suiCompleteRegistrationTxRes, null, 2));
    expect(suiCompleteRegistrationTxRes.effects?.status.status).toBe("failure"); // fails because mock VAA is not from registered emitter

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
