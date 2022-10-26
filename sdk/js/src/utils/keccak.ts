import { ethers } from "ethers";

export function keccak256(data: ethers.BytesLike): Buffer {
  return Buffer.from(ethers.utils.arrayify(ethers.utils.keccak256(data)));
}
