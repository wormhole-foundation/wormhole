/**
 * Precompiled bytecode for the Wormhole wrapped-coin Move module.
 *
 * Creating a wrapped asset on Sui publishes a fresh single-module coin package.
 * Rather than invoke the Move compiler at attestation time, we ship the compiled
 * module and splice in the only two values it is parametrized by: the original
 * token bridge package ID (into the module's address table) and the coin
 * decimals (into the `init` function). The bytecode is otherwise fixed.
 *
 * Ported verbatim from `@certusone/wormhole-sdk` (`sdk/js/src/sui/publish.ts`).
 * To regenerate if the module ever changes: compile the `wrapped_coin` Move
 * template, hex-encode the module, and re-split the output around the token
 * bridge address and the decimals byte into the three fragments below.
 */

/** Decimals are capped at 8 for Sui wrapped assets. */
const MAX_WRAPPED_COIN_DECIMALS = 8;

/** Compiled module up to (and excluding) the spliced token bridge package ID. */
const WRAPPED_COIN_BYTECODE_PREFIX =
  "a11ceb0b060000000901000a020a14031e1704350405392d07669f01088502600ae502050cea02160004010b010c0205020d000002000201020003030c020001000104020700000700010001090801010c020a050600030803040202000302010702080007080100020800080303090002070801010b020209000901010608010105010b0202080008030209000504434f494e095478436f6e7465787408565f5f305f325f3011577261707065644173736574536574757004636f696e0e6372656174655f777261707065640b64756d6d795f6669656c6404696e697414707265706172655f726567697374726174696f6e0f7075626c69635f7472616e736665720673656e646572087472616e736665720a74785f636f6e746578740f76657273696f6e5f636f6e74726f6c00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002";

/** Fixed bytes between the token bridge package ID and the decimals byte. */
const WRAPPED_COIN_BYTECODE_BEFORE_DECIMALS =
  "00020106010000000001090b0031";

/** Fixed bytes following the decimals byte (the remainder of `init`). */
const WRAPPED_COIN_BYTECODE_AFTER_DECIMALS =
  "0a0138000b012e110238010200";

/**
 * Assemble the base64-encoded wrapped-coin module for a given token bridge
 * package and decimals.
 *
 * @param originalTokenBridgePackageId The original (type-origin) token bridge
 *   package ID, with or without a `0x` prefix.
 * @param decimals Desired coin decimals; capped at {@link MAX_WRAPPED_COIN_DECIMALS}.
 * @returns The compiled module bytecode, base64-encoded.
 */
export const buildWrappedCoinBytecode = (
  originalTokenBridgePackageId: string,
  decimals: number
): string => {
  const strippedPackageId = originalTokenBridgePackageId.replace(/^0x/, "");
  const cappedDecimals = Math.min(decimals, MAX_WRAPPED_COIN_DECIMALS);
  const bytecodeHex =
    WRAPPED_COIN_BYTECODE_PREFIX +
    strippedPackageId +
    WRAPPED_COIN_BYTECODE_BEFORE_DECIMALS +
    cappedDecimals.toString(16).padStart(2, "0") +
    WRAPPED_COIN_BYTECODE_AFTER_DECIMALS;
  return Buffer.from(bytecodeHex, "hex").toString("base64");
};
