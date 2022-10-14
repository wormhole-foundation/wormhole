import { hexZeroPad } from "ethers/lib/utils";
import { sha3_256 } from "js-sha3";
import { ChainId, CHAIN_ID_APTOS, ensureHexPrefix, hex } from "../utils";

export const getAssetFullyQualifiedType = (
  tokenBridgeAddress: string, // 32 bytes
  originChain: ChainId,
  originAddress: string
): string | null => {
  // native asset
  if (originChain === CHAIN_ID_APTOS) {
    // originAddress should be of form address::module::type
    if (/(0x)?[0-9a-fA-F]+::\w+::\w+/g.test(originAddress)) {
      console.error("Need fully qualified address for native asset");
      return null;
    }

    return ensureHexPrefix(originAddress);
  }

  // non-native asset, derive unique address
  const wrappedAssetAddress = getForeignAssetAddress(
    tokenBridgeAddress,
    originChain,
    originAddress
  );
  return wrappedAssetAddress ? `${ensureHexPrefix(wrappedAssetAddress)}::coin::T` : null;
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
