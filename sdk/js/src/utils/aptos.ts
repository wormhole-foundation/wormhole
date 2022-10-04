import { hexZeroPad } from "ethers/lib/utils";
import { sha3_256 } from "js-sha3";
import { ChainId, CHAIN_ID_APTOS, hex } from "../utils";

export const getAssetFullyQualifiedType = (
  tokenBridgeAddress: string, // 32 bytes
  originChain: ChainId,
  originAddress: string
): string | null => {
  // native asset
  if (originChain === CHAIN_ID_APTOS) {
    // originAddress should be of form address::module::type
    if (/(0[xX])?[0-9a-fA-F]+::\w+::\w+/g.test(originAddress)) {
      console.error("Need fully qualified address for native asset");
      return null;
    }

    return originAddress;
  }

  // non-native asset, derive unique address
  const wrappedAssetAddress = getForeignAssetAddress(
    tokenBridgeAddress,
    originChain,
    originAddress
  );
  const ensureHexPrefixAddress =
    wrappedAssetAddress!.substring(0, 2).toLowerCase() !== "0x"
      ? `0x${wrappedAssetAddress}`
      : wrappedAssetAddress;
  return `${ensureHexPrefixAddress}::coin::T`;
};

export const getForeignAssetAddress = (
  tokenBridgeAddress: string, // 32 bytes
  originChain: ChainId,
  originAddress: string
): string | null => {
  if (originChain === CHAIN_ID_APTOS) {
    return null;
  }

  let chain: Buffer = Buffer.alloc(2);
  chain.writeUInt16BE(originChain);
  return sha3_256(
    Buffer.concat([
      hex(hexZeroPad(tokenBridgeAddress, 32)),
      chain,
      Buffer.from("::", "ascii"),
      hex(hexZeroPad(originAddress, 32)),
    ])
  );
};
