import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  humanAddress,
} from "@certusone/wormhole-sdk";
import { PublicKey } from "@solana/web3.js";
import { hexValue, hexZeroPad } from "ethers/lib/utils";

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
      : c === CHAIN_ID_ETH
      ? hexZeroPad(hexValue(hexToUint8Array(h)), 20)
      : c === CHAIN_ID_TERRA
      ? humanAddress(hexToUint8Array(h.substr(24))) // terra expects 20 bytes, not 32
      : h;
  } catch (e) {}
  return undefined;
};
