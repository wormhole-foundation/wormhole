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
  fromB64,
  getMoveObjectType,
  JsonRpcProvider,
  RawSigner,
} from "@mysten/sui.js";
import {
  CONTRACTS,
  executeTransactionBlock,
  getInnerType,
  SUI_OBJECT_IDS,
} from "../../utils";
import { attestFromSui } from "../attest";
import { assertIsNotNullOrUndefined } from "./helpers";
import { SUI_FAUCET_URL, SUI_NODE_URL } from "./consts";

const JEST_TEST_TIMEOUT = 60000;
jest.setTimeout(JEST_TEST_TIMEOUT);

const SUI_CORE_BRIDGE_ADDRESS = CONTRACTS.DEVNET.sui.core;
const SUI_TOKEN_BRIDGE_ADDRESS = CONTRACTS.DEVNET.sui.token_bridge;
const SUI_CORE_STATE_OBJECT_ID = SUI_OBJECT_IDS.DEVNET.core_state;
const SUI_TOKEN_BRIDGE_STATE_OBJECT_ID =
  SUI_OBJECT_IDS.DEVNET.token_bridge_state;
const SUI_DEPLOYER_PRIVATE_KEY = "AGA20wtGcwbcNAG4nwapbQ5wIuXwkYQEWFUoSVAxctHb";

// const ethProvider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
// const ethSigner = new ethers.Wallet(ETH_PRIVATE_KEY, ethProvider);

let suiKeypair: Ed25519Keypair;
let suiAddress: string;
let suiProvider: JsonRpcProvider;
let suiSigner: RawSigner;

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
    // const SUI_COIN_TYPE = "0x2::sui::SUI";

    // Get COIN_8 coin type
    const res = await suiProvider.getOwnedObjects({
      owner: suiAddress,
      options: { showContent: true, showType: true },
    });
    console.log(JSON.stringify(res.data, null, 2));
    const coins = res.data.filter((o) =>
      (o.data?.type ?? "").includes("COIN_8")
    );
    expect(coins.length).toBe(1);

    const coin8 = coins[0];
    const coin8Type = getInnerType(getMoveObjectType(coin8) ?? "");
    const coin8TreasuryCapObjectId = coin8.data?.objectId;
    assertIsNotNullOrUndefined(coin8Type);
    assertIsNotNullOrUndefined(coin8TreasuryCapObjectId);

    const suiAttestTxPayload = await attestFromSui(
      suiProvider,
      SUI_TOKEN_BRIDGE_ADDRESS,
      SUI_CORE_STATE_OBJECT_ID,
      SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
      coin8Type,
      0
    );
    const suiAttestTxResult = await executeTransactionBlock(
      suiSigner,
      suiAttestTxPayload
    );
    expect(suiAttestTxResult.effects?.status.status).toBe("success");

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
