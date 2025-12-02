import { BasePeer } from "./types.js";
import { ethers } from "ethers";

export function hashPeerData(basePeer: BasePeer): string {
  return ethers.keccak256(
    ethers.solidityPacked(
      ['string', 'string'],
      [`${basePeer.hostname}:${basePeer.port}`, basePeer.tlsX509]
    )
  );
}