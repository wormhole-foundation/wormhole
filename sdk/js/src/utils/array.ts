import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  CHAIN_ID_POLYGON,
  CHAIN_ID_ETHEREUM_ROPSTEN,
  CHAIN_ID_AVAX,
} from "./consts";
import { humanAddress, canonicalAddress, isNativeDenom } from "../terra";
import { PublicKey } from "@solana/web3.js";
import { hexValue, hexZeroPad, stripZeros } from "ethers/lib/utils";
import { arrayify, zeroPad } from "@ethersproject/bytes";

export const isEVMChain = (chainId: ChainId) => {
  return (
    chainId === CHAIN_ID_ETH ||
    chainId === CHAIN_ID_BSC ||
    chainId === CHAIN_ID_ETHEREUM_ROPSTEN ||
    chainId === CHAIN_ID_AVAX ||
    chainId === CHAIN_ID_POLYGON
  );
};

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
      : isEVMChain(c)
      ? hexZeroPad(hexValue(hexToUint8Array(h)), 20)
      : c === CHAIN_ID_TERRA
      ? isHexNativeTerra(h)
        ? nativeTerraHexToDenom(h)
        : humanAddress(hexToUint8Array(h.substr(24))) // terra expects 20 bytes, not 32
      : h;
  } catch (e) {}
  return undefined;
};

export const nativeToHexString = (
  address: string | undefined,
  chain: ChainId
) => {
  if (!address || !chain) {
    return null;
  }

  if (isEVMChain(chain)) {
    return uint8ArrayToHex(zeroPad(arrayify(address), 32));
  } else if (chain === CHAIN_ID_SOLANA) {
    return uint8ArrayToHex(zeroPad(new PublicKey(address).toBytes(), 32));
  } else if (chain === CHAIN_ID_TERRA) {
    if (isNativeDenom(address)) {
      return (
        "01" +
        uint8ArrayToHex(
          zeroPad(new Uint8Array(Buffer.from(address, "ascii")), 31)
        )
      );
    } else {
      return uint8ArrayToHex(zeroPad(canonicalAddress(address), 32));
    }
  } else {
    return null;
  }
};

export const uint8ArrayToNative = (a: Uint8Array, chainId: ChainId) =>
  hexToNativeString(uint8ArrayToHex(a), chainId);
