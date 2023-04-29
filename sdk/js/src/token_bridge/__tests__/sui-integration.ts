import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import {
  afterAll,
  beforeAll,
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
  getMoveObjectType,
  getPublishedObjectChanges,
} from "@mysten/sui.js";
import { ethers } from "ethers";
import { parseUnits } from "ethers/lib/utils";
import {
  approveEth,
  attestFromEth,
  attestFromSui,
  createWrappedOnEth,
  createWrappedOnSui,
  createWrappedOnSuiPrepare,
  getEmitterAddressEth,
  getForeignAssetEth,
  getForeignAssetSui,
  getIsTransferCompletedEth,
  getIsTransferCompletedSui,
  getIsWrappedAssetSui,
  getOriginalAssetSui,
  getSignedVAAWithRetry,
  parseAttestMetaVaa,
  parseSequenceFromLogEth,
  redeemOnEth,
  redeemOnSui,
  transferFromEth,
  transferFromSui,
} from "../..";
import { executeTransactionBlock, getInnerType, getPackageId } from "../../sui";
import {
  CHAIN_ID_ETH,
  CHAIN_ID_SUI,
  CONTRACTS,
  hexToUint8Array,
  tryNativeToUint8Array,
} from "../../utils";
import { Payload, VAA, parse, serialiseVAA } from "../../vaa/generic";
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY10,
  SUI_FAUCET_URL,
  SUI_NODE_URL,
  TEST_ERC20,
  WORMHOLE_RPC_HOSTS,
} from "./utils/consts";
import {
  assertIsNotNullOrUndefined,
  getEmitterAddressAndSequenceFromResponseSui,
  mintAndTransferCoinSui,
} from "./utils/helpers";

jest.setTimeout(60000);

// Sui constants
const SUI_CORE_BRIDGE_STATE_OBJECT_ID = CONTRACTS.DEVNET.sui.core;
const SUI_TOKEN_BRIDGE_STATE_OBJECT_ID = CONTRACTS.DEVNET.sui.token_bridge;
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

let suiCoreBridgePackageId: string;
let suiTokenBridgePackageId: string;

beforeAll(async () => {
  suiCoreBridgePackageId = await getPackageId(
    suiProvider,
    SUI_CORE_BRIDGE_STATE_OBJECT_ID
  );
  suiTokenBridgePackageId = await getPackageId(
    suiProvider,
    SUI_TOKEN_BRIDGE_STATE_OBJECT_ID
  );
});

afterAll(async () => {
  await ethProvider.destroy();
});

// Modify the VAA to only have 1 guardian signature
// TODO: remove this when we can deploy the devnet core contract
// deterministically with multiple guardians in the initial guardian set
// Currently the core contract is setup with only 1 guardian in the set
function sliceVAASignatures(vaa: Uint8Array) {
  const parsedVAA = parse(Buffer.from([...vaa]));
  parsedVAA.guardianSetIndex = 0;
  parsedVAA.signatures = [parsedVAA.signatures[0]];
  return hexToUint8Array(serialiseVAA(parsedVAA as VAA<Payload>));
}

describe("Sui SDK tests", () => {
  test.skip("Test prebuilt coin build output", async () => {
    // const vaa =
    //   "0100000000010026ff86c07ef853ef955a63c58a8d08eeb2ac232b91e725bd41baeb3c05c5c18d07aef3c02dc3d5ca8ad0600a447c3d55386d0a0e85b23378d438fbb1e207c3b600000002c3a86f000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000001020000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a000212544b4e0000000000000000000000000000000000000000000000000000000000457468657265756d205465737420546f6b656e00000000000000000000000000";
    // const build = getCoinBuildOutput(
    //   suiCoreBridgePackageId,
    //   suiTokenBridgePackageId,
    //   vaa
    // );
    // const buildManual = await getCoinBuildOutputManual(
    //   "DEVNET",
    //   suiCoreBridgePackageId,
    //   suiTokenBridgePackageId,
    //   vaa
    // );
    // expect(build).toMatchObject(buildManual);
    // expect(buildManual).toMatchObject(build);
  });
  test.skip("Transfer native ERC-20 from Ethereum to Sui and back", async () => {
    // Attest on Ethereum
    const ethAttestTxRes = await attestFromEth(
      ETH_TOKEN_BRIDGE_ADDRESS,
      ethSigner,
      TEST_ERC20
    );

    // Get attest VAA
    const attestSequence = parseSequenceFromLogEth(
      ethAttestTxRes,
      ETH_CORE_BRIDGE_ADDRESS
    );
    expect(attestSequence).toBeTruthy();
    const { vaaBytes: attestVAA } = await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      CHAIN_ID_ETH,
      getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS),
      attestSequence,
      {
        transport: NodeHttpTransport(),
      },
      1000,
      5
    );
    expect(attestVAA).toBeTruthy();

    // Start create wrapped on Sui
    const suiPrepareRegistrationTxPayload = await createWrappedOnSuiPrepare(
      suiCoreBridgePackageId,
      suiTokenBridgePackageId,
      attestVAA,
      suiAddress
    );
    const suiPrepareRegistrationTxRes = await executeTransactionBlock(
      suiSigner,
      suiPrepareRegistrationTxPayload
    );
    // expect(coins.length).toBe(1);
    console.log(JSON.stringify(suiPrepareRegistrationTxRes));
    expect(suiPrepareRegistrationTxRes.effects?.status.status).toBe("success");

    // Complete create wrapped on Sui
    const publishEvents = getPublishedObjectChanges(
      suiPrepareRegistrationTxRes
    );
    expect(publishEvents.length).toBe(1);
    const coinPackageId = publishEvents[0].packageId;
    const suiCompleteRegistrationTxPayload = await createWrappedOnSui(
      suiProvider,
      suiTokenBridgePackageId,
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

    // Get foreign asset
    const foreignAsset = await getForeignAssetSui(
      suiProvider,
      SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
      CHAIN_ID_ETH,
      hexToUint8Array(TEST_ERC20)
    );
    console.log(JSON.stringify(foreignAsset));
    //expect(
    //  await getIsWrappedAssetSui(
    //    suiProvider,
    //    SUI_TOKEN_BRIDGE_ADDRESS,
    //    SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
    //    getWrappedCoinType(coinPackageId)
    //  )
    //).toBe(true);
    //const originalAsset = await getOriginalAssetSui(
    //  suiProvider,
    //  SUI_TOKEN_BRIDGE_ADDRESS,
    //  SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
    //  getWrappedCoinType(coinPackageId)
    //);
    //expect(originalAsset).toMatchObject({
    //  isWrapped: true,
    //  chainId: CHAIN_ID_ETH,
    //  assetAddress: Buffer.from(TEST_ERC20, "hex"),
    //});
    //console.log(originalAsset, Buffer.from(TEST_ERC20, "hex"));
    //const transferAmount = formatUnits(1, 18);
    //// Transfer to Sui
    //await approveEth(
    //  CONTRACTS.DEVNET.ethereum.token_bridge,
    //  TEST_ERC20,
    //  ethSigner,
    //  transferAmount
    //);
    //const transferReceipt = await transferFromEth(
    //  CONTRACTS.DEVNET.ethereum.token_bridge,
    //  ethSigner,
    //  TEST_ERC20,
    //  transferAmount,
    //  CHAIN_ID_SUI,
    //  tryNativeToUint8Array(suiAddress, CHAIN_ID_SUI)
    //);
    //const ethSequence = parseSequenceFromLogEth(
    //  transferReceipt,
    //  ETH_CORE_BRIDGE_ADDRESS
    //);
    //const { vaaBytes: ethTransferVAA } = await getSignedVAAWithRetry(
    //  WORMHOLE_RPC_HOSTS,
    //  CHAIN_ID_ETH,
    //  getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS),
    //  ethSequence,
    //  {
    //    transport: NodeHttpTransport(),
    //  },
    //  1000,
    //  5
    //);
    //expect(ethTransferVAA).toBeTruthy();
    //// Redeem on Sui
    //const redeemPayload = await redeemOnSui(
    //  suiProvider,
    //  SUI_TOKEN_BRIDGE_ADDRESS,
    //  SUI_CORE_BRIDGE_STATE_OBJECT_ID,
    //  SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
    //  ethTransferVAA
    //);
    //await executeTransactionBlock(suiSigner, redeemPayload);
    //expect(
    //  await getIsTransferCompletedSui(
    //    suiProvider,
    //    SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
    //    ethTransferVAA
    //  )
    //).toBe(true);
    //// Transfer back to Eth
    //const coinType = await getForeignAssetSui(
    //  suiProvider,
    //  SUI_TOKEN_BRIDGE_ADDRESS,
    //  SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
    //  CHAIN_ID_ETH,
    //  originalAsset.assetAddress
    //);
    //expect(coinType).toBeTruthy();
    //const coins = (
    //  await suiProvider.getCoins({ owner: suiAddress, coinType })
    //).data.map<SuiCoinObject>((c) => ({
    //  type: c.coinType,
    //  objectId: c.coinObjectId,
    //}));
    //const suiTransferTxPayload = transferFromSui(
    //  SUI_TOKEN_BRIDGE_ADDRESS,
    //  SUI_CORE_BRIDGE_STATE_OBJECT_ID,
    //  SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
    //  coins,
    //  coinType || "",
    //  BigInt(transferAmount),
    //  CHAIN_ID_ETH,
    //  tryNativeToUint8Array(ethSigner.address, CHAIN_ID_ETH)
    //);
    //const suiTransferTxResult = await executeTransactionBlock(
    //  suiSigner,
    //  suiTransferTxPayload
    //);
    //const { sequence, emitterAddress } =
    //  getEmitterAddressAndSequenceFromResponseSui(
    //    SUI_CORE_BRIDGE_ADDRESS,
    //    suiTransferTxResult
    //  );
    //// Fetch the transfer VAA
    //const { vaaBytes: transferVAA } = await getSignedVAAWithRetry(
    //  WORMHOLE_RPC_HOSTS,
    //  CHAIN_ID_SUI,
    //  emitterAddress,
    //  sequence,
    //  {
    //    transport: NodeHttpTransport(),
    //  },
    //  1000,
    //  5
    //);
    //expect(transferVAA).toBeTruthy();
    //// Redeem on Ethereum
    //await redeemOnEth(ETH_TOKEN_BRIDGE_ADDRESS, ethSigner, transferVAA);
    //expect(
    //  await getIsTransferCompletedEth(
    //    ETH_TOKEN_BRIDGE_ADDRESS,
    //    ethProvider,
    //    transferVAA
    //  )
    //).toBe(true);
  });
  test("Transfer non-SUI Sui token to Ethereum and back", async () => {
    // Get COIN_8 coin type
    const res = await suiProvider.getOwnedObjects({
      owner: suiAddress,
      options: { showContent: true, showType: true },
    });
    const coins = res.data.filter((o) => {
      const type = o.data?.type ?? "";
      return type.includes("TreasuryCap") && type.includes("COIN_8");
    });
    expect(coins.length).toBeGreaterThan(0);

    const coin8 = coins[0];
    const coin8Type = getInnerType(getMoveObjectType(coin8) ?? "");
    const coin8TreasuryCapObjectId = coin8.data?.objectId;
    assertIsNotNullOrUndefined(coin8Type);
    assertIsNotNullOrUndefined(coin8TreasuryCapObjectId);
    expect(
      await getIsWrappedAssetSui(
        suiProvider,
        SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
        coin8Type
      )
    ).toBe(false);

    // Mint coins
    const transferAmount = parseUnits("1", 8).toBigInt();
    const suiMintTxPayload = mintAndTransferCoinSui(
      coin8TreasuryCapObjectId,
      coin8Type,
      transferAmount,
      suiAddress
    );
    let result = await executeTransactionBlock(suiSigner, suiMintTxPayload);
    expect(result.effects?.status.status).toBe("success");

    // Attest on Sui
    const suiAttestTxPayload = await attestFromSui(
      suiProvider,
      SUI_CORE_BRIDGE_STATE_OBJECT_ID,
      SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
      coin8Type
    );
    result = await executeTransactionBlock(suiSigner, suiAttestTxPayload);
    expect(result.effects?.status.status).toBe("success");
    const { sequence: attestSequence, emitterAddress: attestEmitterAddress } =
      getEmitterAddressAndSequenceFromResponseSui(
        suiCoreBridgePackageId,
        result
      );
    expect(attestSequence).toBeTruthy();
    expect(attestEmitterAddress).toBeTruthy();
    const { vaaBytes } = await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      CHAIN_ID_SUI,
      attestEmitterAddress,
      attestSequence,
      {
        transport: NodeHttpTransport(),
      },
      1000,
      30
    );
    expect(vaaBytes).toBeTruthy();
    // Create wrapped on Ethereum
    try {
      await createWrappedOnEth(ETH_TOKEN_BRIDGE_ADDRESS, ethSigner, vaaBytes);
    } catch (e) {
      // this could fail because the token is already attested (in an unclean env)
    }
    const { tokenAddress } = parseAttestMetaVaa(vaaBytes);
    expect(
      await getOriginalAssetSui(
        suiProvider,
        SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
        coin8Type
      )
    ).toMatchObject({
      isWrapped: false,
      chainId: CHAIN_ID_SUI,
      assetAddress: new Uint8Array(tokenAddress),
    });
    const coin8Coins = await suiProvider.getCoins({
      owner: suiAddress,
      coinType: coin8Type,
    });
    expect(coin8Coins.data.length).toBeGreaterThan(0);
    // Transfer to Ethereum
    const suiTransferTxPayload = await transferFromSui(
      suiProvider,
      SUI_CORE_BRIDGE_STATE_OBJECT_ID,
      SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
      coin8Coins.data,
      coin8Type,
      transferAmount,
      CHAIN_ID_ETH,
      tryNativeToUint8Array(ethSigner.address, CHAIN_ID_ETH)
    );
    result = await executeTransactionBlock(suiSigner, suiTransferTxPayload);
    expect(result.effects?.status.status).toBe("success");
    const { sequence, emitterAddress } =
      getEmitterAddressAndSequenceFromResponseSui(
        suiCoreBridgePackageId,
        result
      );
    expect(sequence).toBeTruthy();
    expect(emitterAddress).toBeTruthy();
    // Fetch the transfer VAA
    const { vaaBytes: transferVAA } = await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      CHAIN_ID_SUI,
      emitterAddress,
      sequence!,
      {
        transport: NodeHttpTransport(),
      },
      1000,
      30
    );
    // Redeem on Ethereum
    await redeemOnEth(ETH_TOKEN_BRIDGE_ADDRESS, ethSigner, transferVAA);
    expect(
      await getIsTransferCompletedEth(
        ETH_TOKEN_BRIDGE_ADDRESS,
        ethProvider,
        transferVAA
      )
    ).toBe(true);

    // Transfer back to Sui
    const ethTokenAddress = await getForeignAssetEth(
      ETH_TOKEN_BRIDGE_ADDRESS,
      ethProvider,
      CHAIN_ID_SUI,
      tokenAddress
    );
    expect(ethTokenAddress).toBeTruthy();
    await approveEth(
      ETH_TOKEN_BRIDGE_ADDRESS,
      ethTokenAddress!,
      ethSigner,
      transferAmount
    );
    const transferReceipt = await transferFromEth(
      ETH_TOKEN_BRIDGE_ADDRESS,
      ethSigner,
      ethTokenAddress!,
      transferAmount,
      CHAIN_ID_SUI,
      tryNativeToUint8Array(suiAddress, CHAIN_ID_SUI)
    );
    const ethSequence = parseSequenceFromLogEth(
      transferReceipt,
      ETH_CORE_BRIDGE_ADDRESS
    );
    expect(ethSequence).toBeTruthy();
    const { vaaBytes: ethTransferVAA } = await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      CHAIN_ID_ETH,
      getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS),
      ethSequence,
      {
        transport: NodeHttpTransport(),
      },
      1000,
      30
    );
    const slicedVAA = sliceVAASignatures(ethTransferVAA);
    // Redeem on Sui
    const redeemPayload = await redeemOnSui(
      suiProvider,
      SUI_CORE_BRIDGE_STATE_OBJECT_ID,
      SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
      slicedVAA
    );
    result = await executeTransactionBlock(suiSigner, redeemPayload);
    expect(result.effects?.status.status).toBe("success");
    expect(
      await getIsTransferCompletedSui(
        suiProvider,
        SUI_TOKEN_BRIDGE_STATE_OBJECT_ID,
        slicedVAA
      )
    ).toBe(true);
  });
});
