import { keccak256 } from "../utils";
import * as elliptic from "elliptic";

export function ethPrivateToPublic(key: string) {
  const ecdsa = new elliptic.ec("secp256k1");
  const publicKey = ecdsa.keyFromPrivate(key).getPublic("hex");
  return keccak256(Buffer.from(publicKey, "hex").subarray(1)).subarray(12);
}

export function ethSignWithPrivate(privateKey: string, hash: Buffer) {
  if (hash.length != 32) {
    throw new Error("hash.length != 32");
  }
  const ecdsa = new elliptic.ec("secp256k1");
  const key = ecdsa.keyFromPrivate(privateKey);
  return key.sign(hash, { canonical: true });
}
