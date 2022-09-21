import { ethers } from "ethers";
import { sha3_256 } from "js-sha3";
import { ChainId } from "../utils";

export const deriveWrappedAssetAddress = (
  tokenBridgeAddress: string, // 32 bytes
  originChain: ChainId,
  originAddress: string // 32 bytes
): string => {
  let chain: Buffer = Buffer.alloc(2);
  chain.writeUInt16BE(originChain);
  return sha3_256(
    Buffer.concat([
      hex(tokenBridgeAddress),
      chain,
      Buffer.from("::", "ascii"),
      hex(originAddress),
    ])
  );
};

export const hex = (x: string): Buffer => {
  return Buffer.from(
    ethers.utils.hexlify(x, { allowMissingPrefix: true }).substring(2),
    "hex"
  );
};
