import { arrayify, zeroPad } from "@ethersproject/bytes";
import { PublicKey } from "@solana/web3.js";
import { hexValue, hexZeroPad, keccak256, sha256 } from "ethers/lib/utils";
import { bech32 } from "bech32";
import {
  ChainId,
  PlatformToChains,
  chainToChainId,
  encoding,
  toChain,
  toChainId,
} from "@wormhole-foundation/sdk-base";
import { UniversalAddress } from "@wormhole-foundation/sdk-definitions";
import {
  chainToNativeDenoms,
  CosmwasmAddress,
} from "@wormhole-foundation/sdk-cosmwasm";
import { isValidSuiAddress } from "./sui";
import { sha3_256 } from "js-sha3";
import {
  nativeStringToHexAlgorand,
  uint8ArrayToNativeStringAlgorand,
} from "@certusone/wormhole-sdk/lib/esm/algorand";
import { isValidSuiType } from "@certusone/wormhole-sdk/lib/esm/sui";
import { ChainId as LegacyChainId } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import {
  TERRA2_ADDRESS_PREFIX,
  TERRA2_NATIVE_DENOM,
  Terra2Like,
  isTerra2Like,
} from "../chains/terra2/consts";
import {
  CliChainLike,
  cliChainToChainId,
  cliChainToPlatform,
  toCliChain,
} from "../utils";

/**
 * Convert a chain to the legacy \@certusone/wormhole-sdk ChainId type.
 *
 * The legacy SDK is frozen, so its ChainId union doesn't include chains added
 * after its last release. On the wire a chain id is just a uint16, so passing
 * newer ids through the legacy functions is safe. Terra2 and the chains the
 * SDK dropped entirely are handled by `cliChainToChainId` (the legacy SDK
 * still knows their ids).
 */
export const toLegacyChainId = (chain: CliChainLike): LegacyChainId =>
  cliChainToChainId(toCliChain(chain)) as number as LegacyChainId;

const isLikely20ByteCosmwasm = (h: string): boolean =>
  h.startsWith("000000000000000000000000");

export const uint8ArrayToHex = (a: Uint8Array): string =>
  encoding.hex.encode(a);

export const hexToUint8Array = (h: string): Uint8Array =>
  encoding.hex.decode(h);

export function canonicalAddress(humanAddress: string) {
  return new Uint8Array(bech32.fromWords(bech32.decode(humanAddress).words));
}

export function humanAddress(
  hrp: string,
  canonicalAddress: Uint8Array
): string {
  return CosmwasmAddress.encode(hrp, canonicalAddress);
}

export function buildTokenId(
  chain: Exclude<PlatformToChains<"Cosmwasm">, "Seda"> | Terra2Like,
  address: string
) {
  // the SDK no longer maps Terra2's native denom; the compat layer does
  const nativeDenom = isTerra2Like(chain)
    ? TERRA2_NATIVE_DENOM
    : chainToNativeDenoms("Mainnet", chain);
  return (
    (nativeDenom === address ? "01" : "00") +
    keccak256(Buffer.from(address, "utf-8")).substring(4)
  );
}

/**
 *
 * Convert an address in a wormhole's 32-byte array representation into a chain's
 * native string representation.
 *
 * @throws if address is not the right length for the given chain
 */

export const tryUint8ArrayToNative = (
  a: Uint8Array,
  chain: CliChainLike
): string => {
  const chainName = toCliChain(chain);
  if (cliChainToPlatform(chainName) === "Evm") {
    return hexZeroPad(hexValue(a), 20);
  } else if (cliChainToPlatform(chainName) === "Solana") {
    return new PublicKey(a).toString();
  } else if (chainName === "Injective") {
    const h = uint8ArrayToHex(a);
    return humanAddress("inj", isLikely20ByteCosmwasm(h) ? a.slice(-20) : a);
  } else if (chainName === "Terra2") {
    const h = uint8ArrayToHex(a);
    if (h.startsWith("01")) {
      // native denoms are encoded with the first byte set to 1
      return Buffer.from(
        hexToUint8Array(h.substring(2)).filter((b) => b !== 0)
      ).toString("ascii");
    }
    // terra2 has 32 byte addresses for contracts and 20 for wallets
    return humanAddress(
      TERRA2_ADDRESS_PREFIX,
      isLikely20ByteCosmwasm(h) ? a.slice(-20) : a
    );
  } else if (chainName === "Algorand") {
    return uint8ArrayToNativeStringAlgorand(a);
  } else if (chainName == "Wormchain") {
    const h = uint8ArrayToHex(a);
    return humanAddress(
      "wormhole",
      isLikely20ByteCosmwasm(h) ? a.slice(-20) : a
    );
  } else if (chainName === "Sei") {
    const h = uint8ArrayToHex(a);
    return humanAddress("sei", isLikely20ByteCosmwasm(h) ? a.slice(-20) : a);
  } else if (chainName === "Near") {
    throw Error("uint8ArrayToNative: Use tryHexToNativeStringNear instead.");
  } else if (chainName === "Osmosis") {
    throw Error("uint8ArrayToNative: Osmosis not supported yet.");
  } else if (chainName === "Cosmoshub") {
    throw Error("uint8ArrayToNative: CosmosHub not supported yet.");
  } else if (chainName === "Evmos") {
    throw Error("uint8ArrayToNative: Evmos not supported yet.");
  } else if (chainName === "Kujira") {
    throw Error("uint8ArrayToNative: Kujira not supported yet.");
  } else if (chainName === "Neutron") {
    throw Error("uint8ArrayToNative: Neutron not supported yet.");
  } else if (chainName === "Celestia") {
    throw Error("uint8ArrayToNative: Celestia not supported yet.");
  } else if (chainName === "Stargaze") {
    throw Error("uint8ArrayToNative: Stargaze not supported yet.");
  } else if (chainName === "Seda") {
    throw Error("uint8ArrayToNative: Seda not supported yet.");
  } else if (chainName === "Dymension") {
    throw Error("uint8ArrayToNative: Dymension not supported yet.");
  } else if (chainName === "Provenance") {
    throw Error("uint8ArrayToNative: Provenance not supported yet.");
  } else if (chainName === "Sui") {
    throw Error("uint8ArrayToNative: Sui not supported yet.");
  } else if (chainName === "Aptos") {
    throw Error("uint8ArrayToNative: Aptos not supported yet.");
  } else if (chainName === "Btc") {
    throw Error("uint8ArrayToNative: Btc not supported");
  } else {
    // This case is never reached
    // const _: never = chainName;
    throw Error("Don't know how to convert address for chain " + chainName);
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
  c === chainToChainId("Algorand")
    ? // Algorand assets are represented by their asset ids, not an address
      new UniversalAddress(h).toNative("Algorand").toBigInt().toString()
    : new UniversalAddress(h).toNative(toChain(c)).toString();

/**
 *
 * Convert an address in a chain's native representation into a 32-byte hex string
 * understood by wormhole (UniversalAddress).
 *
 * @throws if address is a malformed string for the given chain id
 */
export const tryNativeToHexString = (
  address: string,
  chain: CliChainLike
): string => {
  const chainName = toCliChain(chain);
  if (cliChainToPlatform(chainName) === "Evm") {
    return uint8ArrayToHex(zeroPad(arrayify(address), 32));
  } else if (cliChainToPlatform(chainName) === "Solana") {
    return uint8ArrayToHex(zeroPad(new PublicKey(address).toBytes(), 32));
  } else if (
    chainName === "Injective" ||
    chainName === "Sei" ||
    chainName === "Terra2"
  ) {
    return buildTokenId(chainName, address);
  } else if (chainName === "Algorand") {
    return nativeStringToHexAlgorand(address);
  } else if (chainName == "Wormchain") {
    return uint8ArrayToHex(zeroPad(canonicalAddress(address), 32));
  } else if (chainName === "Near") {
    return uint8ArrayToHex(arrayify(sha256(Buffer.from(address))));
  } else if (chainName === "Sui") {
    if (!isValidSuiType(address) && isValidSuiAddress(address)) {
      return uint8ArrayToHex(
        zeroPad(arrayify(address, { allowMissingPrefix: true }), 32)
      );
    }
    throw Error("nativeToHexString: Sui types not supported yet.");
  } else if (chainName === "Aptos") {
    if (isValidAptosType(address)) {
      return getExternalAddressFromType(address);
    }

    return uint8ArrayToHex(
      zeroPad(arrayify(address, { allowMissingPrefix: true }), 32)
    );
  } else {
    // If this case is reached
    throw Error(`nativeToHexString: ${chainName} not supported yet.`);
  }
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
  chain: CliChainLike
): Uint8Array {
  return hexToUint8Array(tryNativeToHexString(address, chain));
}

/**
 * Test if given string is a valid fully qualified type of moduleAddress::moduleName::structName.
 * @param str String to test
 * @returns Whether or not given string is a valid type
 */
export const isValidAptosType = (str: string): boolean =>
  /^(0x)?[0-9a-fA-F]+::\w+::\w+$/.test(str);

/**
 * Hashes the given type. Because fully qualified types are a concept unique to Aptos, this
 * output acts as the address on other chains.
 * @param fullyQualifiedType Fully qualified type on Aptos
 * @returns External address corresponding to given type
 */
export const getExternalAddressFromType = (
  fullyQualifiedType: string
): string => {
  // hash the type so it fits into 32 bytes
  return sha3_256(fullyQualifiedType);
};
