import { zeroPad } from "@ethersproject/bytes";
import { bech32 } from "bech32";
import { ADDRESS_PREFIX } from "../consts";

export const uint8ArrayToHex = (a: Uint8Array) =>
  Buffer.from(a).toString("hex");

export const hexToUint8Array = (h: string) =>
  new Uint8Array(Buffer.from(h, "hex"));

export function canonicalAddress(humanAddress: string) {
    return new Uint8Array(bech32.fromWords(bech32.decode(humanAddress).words));
}

export function humanAddress(canonicalAddress: Uint8Array) {
    return bech32.encode(ADDRESS_PREFIX, bech32.toWords(canonicalAddress));
  }

export function nativeToHexAddress(nativeAddress : string) {
    return uint8ArrayToHex(zeroPad(canonicalAddress(nativeAddress), 32))
}

export function hexToNativeAddress(hexAddress : string) {
    return humanAddress(hexToUint8Array(hexAddress.substr(24)))
}