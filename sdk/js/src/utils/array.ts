import { arrayify, zeroPad } from "@ethersproject/bytes";
import { PublicKey } from "@solana/web3.js";
import { hexValue, hexZeroPad, stripZeros } from "ethers/lib/utils";
import {
  hexToNativeAssetStringAlgorand,
  nativeStringToHexAlgorand,
  uint8ArrayToNativeStringAlgorand,
} from "../algorand";
import { canonicalAddress, humanAddress, isNativeDenom } from "../terra";
import {
  ChainId,
  CHAIN_ID_ALGORAND,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  isEVMChain,
} from "./consts";

/**
 *
 * Returns true iff the hex string represents a native Terra denom.
 *
 * Native assets on terra don't have an associated smart contract address, just
 * like eth isn't an ERC-20 contract on Ethereum.
 *
 * The difference is that the EVM implementations of Portal don't support eth
 * directly, and instead require swapping to an ERC-20 wrapped eth (WETH)
 * contract first.
 *
 * The Terra implementation instead supports Terra-native denoms without
 * wrapping to CW-20 token first. As these denoms don't have an address, they
 * are encoded in the Portal payloads by the setting the first byte to 1.  This
 * encoding is safe, because the first 12 bytes of the 32-byte wormhole address
 * space are not used on Terra otherwise, as cosmos addresses are 20 bytes wide.
 */
export const isHexNativeTerra = (h: string): boolean => h.startsWith("01");

export const nativeTerraHexToDenom = (h: string): string =>
  Buffer.from(stripZeros(hexToUint8Array(h.substr(2)))).toString("ascii");

export const uint8ArrayToHex = (a: Uint8Array): string =>
  Buffer.from(a).toString("hex");

export const hexToUint8Array = (h: string): Uint8Array =>
  new Uint8Array(Buffer.from(h, "hex"));

/**
 *
 * Convert an address in a wormhole's 32-byte array representation into a chain's
 * native string representation.
 *
 * @throws if address is not the right length for the given chain
 */
export const tryUint8ArrayToNative = (
  a: Uint8Array,
  chainId: ChainId
): string => {
  if (isEVMChain(chainId)) {
    return hexZeroPad(hexValue(a), 20);
  } else if (chainId === CHAIN_ID_SOLANA) {
    return new PublicKey(a).toString();
  } else if (chainId === CHAIN_ID_TERRA) {
    const h = uint8ArrayToHex(a);
    if (isHexNativeTerra(h)) {
      return nativeTerraHexToDenom(h);
    } else {
      return humanAddress(a.slice(-20)); // terra expects 20 bytes, not 32
    }
  } else if (chainId === CHAIN_ID_ALGORAND) {
    return uint8ArrayToNativeStringAlgorand(a);
  } else {
    // This case is never reached
    const _: never = chainId;
    throw Error("Don't know how to convert address for chain " + chainId);
  }
};

/**
 *
 * Convert an address in a wormhole's 32-byte hex representation into a chain's native
 * string representation.
 *
 * @throws if address is not the right length for the given chain
 */
export const tryHexToNativeAssetString = (h: string, c: ChainId): string =>
  c === CHAIN_ID_ALGORAND
    ? // Algorand assets are represented by their asset ids, not an address
      hexToNativeAssetStringAlgorand(h)
    : tryHexToNativeString(h, c);

/**
 *
 * Convert an address in a wormhole's 32-byte hex representation into a chain's native
 * string representation.
 *
 * @deprecated since 0.2.8, use [[tryHexToNativeString]] instead.
 */
export const hexToNativeAssetString = (
  h: string | undefined,
  c: ChainId
): string | undefined => {
  if (!h) {
    return undefined;
  }
  try {
    return tryHexToNativeAssetString(h, c);
  } catch (e) {
    return undefined;
  }
};

/**
 *
 * Convert an address in a wormhole's 32-byte hex representation into a chain's native
 * string representation.
 *
 * @throws if address is not the right length for the given chain
 */
export const tryHexToNativeString = (h: string, c: ChainId): string =>
  tryUint8ArrayToNative(hexToUint8Array(h), c);

/**
 *
 * Convert an address in a wormhole's 32-byte hex representation into a chain's native
 * string representation.
 *
 * @deprecated since 0.2.8, use [[tryHexToNativeString]] instead.
 */
export const hexToNativeString = (
  h: string | undefined,
  c: ChainId
): string | undefined => {
  if (!h) {
    return undefined;
  }

  try {
    return tryHexToNativeString(h, c);
  } catch (e) {
    return undefined;
  }
};

/**
 *
 * Convert an address in a chain's native representation into a 32-byte hex string
 * understood by wormhole.
 *
 * @throws if address is a malformed string for the given chain id
 */
export const tryNativeToHexString = (
  address: string,
  chain: ChainId
): string => {
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
  } else if (chain === CHAIN_ID_ALGORAND) {
    return nativeStringToHexAlgorand(address);
  } else {
    // If this case is reached
    const _: never = chain;
    throw Error("Don't know how to convert address from chain " + chain);
  }
};

/**
 *
 * Convert an address in a chain's native representation into a 32-byte hex string
 * understood by wormhole.
 *
 * @deprecated since 0.2.8, use [[tryNativeToHexString]] instead.
 * @throws if address is a malformed string for the given chain id
 */
export const nativeToHexString = (
  address: string | undefined,
  chain: ChainId
): string | null => {
  if (!address) {
    return null;
  }
  return tryNativeToHexString(address, chain);
};

/**
 *
 * Convert an address in a chain's native representation into a 32-byte array
 * understood by wormhole.
 *
 * @throws if address is a malformed string for the given chain id
 */
export function tryNativeToUint8Array(
  address: string,
  chainId: ChainId
): Uint8Array {
  return hexToUint8Array(tryNativeToHexString(address, chainId));
}

export function chunks<T>(array: T[], size: number): T[][] {
  return Array.apply<number, T[], T[][]>(
    0,
    new Array(Math.ceil(array.length / size))
  ).map((_, index) => array.slice(index * size, (index + 1) * size));
}

export function textToHexString(name: string): string {
  return Buffer.from(name, "binary").toString("hex");
}

export function textToUint8Array(name: string): Uint8Array {
  return new Uint8Array(Buffer.from(name, "binary"));
}
