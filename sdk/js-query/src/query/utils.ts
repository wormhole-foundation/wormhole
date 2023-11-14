import { Buffer } from "buffer";
import * as elliptic from "elliptic";

export function isValidHexString(s: string): boolean {
  return /^(0x)?[0-9a-fA-F]+$/.test(s);
}

export function hexToUint8Array(s: string): Uint8Array {
  if (!isValidHexString(s)) {
    throw new Error(`${s} is not hex`);
  }
  if (s.startsWith("0x")) {
    s = s.slice(2);
  }
  s.padStart(s.length + (s.length % 2), "0");
  return new Uint8Array(Buffer.from(s, "hex"));
}

export function uint8ArrayToHex(b: Uint8Array) {
  return `0x${Buffer.from(b).toString("hex")}`;
}

export function coalesceUint8Array(b: string | Uint8Array): Uint8Array {
  return typeof b === "string" ? hexToUint8Array(b) : b;
}

/**
 * @param key Private key used to sign `data`
 * @param data Data for signing
 * @returns ECDSA secp256k1 signature
 */
export function sign(key: string, data: Uint8Array): string {
  const ec = new elliptic.ec("secp256k1");
  const keyPair = ec.keyFromPrivate(key);
  const signature = keyPair.sign(data, { canonical: true });
  const packed =
    signature.r.toString("hex").padStart(64, "0") +
    signature.s.toString("hex").padStart(64, "0") +
    Buffer.from([signature.recoveryParam ?? 0]).toString("hex");
  return packed;
}
