import {
  fromB64,
  JsonRpcProvider,
  normalizeSuiObjectId,
  TransactionBlock,
} from "@mysten/sui.js";
import { SuiBuildOutput } from "./types";
import { getOriginalPackageId, getPackageId } from "./utils";

export const publishCoin = async (
  provider: JsonRpcProvider,
  coreBridgeStateObjectId: string,
  tokenBridgeStateObjectId: string,
  decimals: number,
  signerAddress: string,
  coreBridgePackageId?: string,
  tokenBridgePackageId?: string
) => {
  [coreBridgePackageId, tokenBridgePackageId] = await Promise.all([
    coreBridgePackageId
      ? Promise.resolve(coreBridgePackageId)
      : getPackageId(provider, coreBridgeStateObjectId),
    tokenBridgePackageId
      ? Promise.resolve(tokenBridgePackageId)
      : getPackageId(provider, tokenBridgeStateObjectId),
  ]);
  const build = await getCoinBuildOutput(
    provider,
    coreBridgePackageId,
    tokenBridgePackageId,
    tokenBridgeStateObjectId,
    decimals
  );
  return publishPackage(build, signerAddress);
};

export const getCoinBuildOutput = async (
  provider: JsonRpcProvider,
  coreBridgePackageId: string,
  tokenBridgePackageId: string,
  tokenBridgeStateObjectId: string,
  decimals: number
): Promise<SuiBuildOutput> => {
  // Decimals is capped at 8
  decimals = Math.min(decimals, 8);

  // Construct bytecode, parametrized by token bridge package ID and decimals
  const strippedTokenBridgePackageId = (
    await getOriginalPackageId(provider, tokenBridgeStateObjectId)
  )?.replace("0x", "");
  if (!strippedTokenBridgePackageId) {
    throw new Error(
      `Original token bridge package ID not found for object ID ${tokenBridgeStateObjectId}`
    );
  }

  const bytecodeHex =
    "a11ceb0b060000000901000a020a14031e1704350405392d07669f01088502600ae502050cea02160004010b010c0205020d000002000201020003030c020001000104020700000700010001090801010c020a050600030803040202000302010702080007080100020800080303090002070801010b020209000901010608010105010b0202080008030209000504434f494e095478436f6e7465787408565f5f305f325f3011577261707065644173736574536574757004636f696e0e6372656174655f777261707065640b64756d6d795f6669656c6404696e697414707265706172655f726567697374726174696f6e0f7075626c69635f7472616e736665720673656e646572087472616e736665720a74785f636f6e746578740f76657273696f6e5f636f6e74726f6c00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002" +
    strippedTokenBridgePackageId +
    "00020106010000000001090b0031" +
    decimals.toString(16).padStart(2, "0") +
    "0a0138000b012e110238010200";
  const bytecode = Buffer.from(bytecodeHex, "hex").toString("base64");
  return {
    modules: [bytecode],
    dependencies: ["0x1", "0x2", tokenBridgePackageId, coreBridgePackageId].map(
      (d) => normalizeSuiObjectId(d)
    ),
  };
};

export const publishPackage = async (
  buildOutput: SuiBuildOutput,
  signerAddress: string
): Promise<TransactionBlock> => {
  // Publish contracts
  const tx = new TransactionBlock();
  const [upgradeCap] = tx.publish({
    modules: buildOutput.modules.map((m) => Array.from(fromB64(m))),
    dependencies: buildOutput.dependencies.map((d) => normalizeSuiObjectId(d)),
  });

  // Transfer upgrade capability to recipient
  tx.transferObjects([upgradeCap], tx.pure(signerAddress));
  return tx;
};
