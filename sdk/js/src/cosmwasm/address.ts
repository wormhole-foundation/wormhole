import { bech32 } from "bech32";
import { keccak256 } from "ethers/lib/utils";
import { isNativeDenom } from "../terra";

export function canonicalAddress(humanAddress: string) {
  return new Uint8Array(bech32.fromWords(bech32.decode(humanAddress).words));
}
export function humanAddress(
  canonicalAddress: Uint8Array,
  prefix: string = "terra"
) {
  return bech32.encode(prefix, bech32.toWords(canonicalAddress));
}

export function buildTokenId(address: string) {
  return (
    (isNativeDenom(address) ? "01" : "00") +
    keccak256(Buffer.from(address, "utf-8")).substring(4)
  );
}
