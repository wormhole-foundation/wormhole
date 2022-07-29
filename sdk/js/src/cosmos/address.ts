import { bech32 } from "bech32";

export function canonicalAddress(humanAddress: string) {
  return new Uint8Array(bech32.fromWords(bech32.decode(humanAddress).words));
}
export function humanAddress(hrp: string, canonicalAddress: Uint8Array) {
  return bech32.encode(hrp, bech32.toWords(canonicalAddress));
}
