import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import { PublicKey } from "@solana/web3.js";
import { hexValue } from "ethers/lib/utils";

export const uint8ArrayToHex = (a: Uint8Array) =>
  Buffer.from(a).toString("hex");
export const hexToUint8Array = (h: string) =>
  new Uint8Array(Buffer.from(h, "hex"));
export const hexToNativeString = (h: string | undefined, c: ChainId) =>
  !h
    ? undefined
    : c === CHAIN_ID_SOLANA
    ? new PublicKey(hexToUint8Array(h)).toString()
    : c === CHAIN_ID_ETH
    ? hexValue(hexToUint8Array(h))
    : h;
