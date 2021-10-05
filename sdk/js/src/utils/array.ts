import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "./consts";
import { humanAddress } from "../terra";
import { PublicKey } from "@solana/web3.js";
import { hexValue, hexZeroPad, stripZeros } from "ethers/lib/utils";

export const isHexNativeTerra = (h: string) => h.startsWith("01");
export const nativeTerraHexToDenom = (h: string) =>
  Buffer.from(stripZeros(hexToUint8Array(h.substr(2)))).toString("ascii");
export const uint8ArrayToHex = (a: Uint8Array) =>
  Buffer.from(a).toString("hex");
export const hexToUint8Array = (h: string) =>
  new Uint8Array(Buffer.from(h, "hex"));
export const hexToNativeString = (h: string | undefined, c: ChainId) => {
  try {
    return !h
      ? undefined
      : c === CHAIN_ID_SOLANA
      ? new PublicKey(hexToUint8Array(h)).toString()
      : c === CHAIN_ID_ETH || c === CHAIN_ID_BSC
      ? hexZeroPad(hexValue(hexToUint8Array(h)), 20)
      : c === CHAIN_ID_TERRA
      ? isHexNativeTerra(h)
        ? nativeTerraHexToDenom(h)
        : humanAddress(hexToUint8Array(h.substr(24))) // terra expects 20 bytes, not 32
      : h;
  } catch (e) {}
  return undefined;
};
