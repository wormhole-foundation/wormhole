import { sha3_256 } from "js-sha3";
import { ChainId, CHAIN_ID_APTOS, hex } from "../utils";

export const getAssetFullyQualifiedType = (
  tokenBridgeAddress: string, // 32 bytes
  originChain: ChainId,
  originAddress: string,
): string => {
  // native asset
  if (originChain === CHAIN_ID_APTOS) {
    // originAddress should be of form address::module::type
    if ((originAddress.match(/::/g) || []).length !== 2) {
      throw "Need fully qualified address for native asset";
    }

    return originAddress;
  }

  // non-native asset, derive unique address
  let chain: Buffer = Buffer.alloc(2);
  chain.writeUInt16BE(originChain);
  const wrappedAssetAddress = sha3_256(
    Buffer.concat([hex(tokenBridgeAddress), chain, Buffer.from("::", "ascii"), hex(originAddress)]),
  );
  return `${wrappedAssetAddress}::coin::T`;
};
