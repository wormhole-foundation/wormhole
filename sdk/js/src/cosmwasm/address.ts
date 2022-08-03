import { bech32 } from "bech32";
import { keccak256 } from "ethers/lib/utils";
import { isNativeDenom } from "../terra";
import {
  CHAIN_ID_INJECTIVE,
  CHAIN_ID_TERRA2,
  CosmWasmChainId,
  isTerraChain,
} from "../utils";

export function canonicalAddress(humanAddress: string) {
  return new Uint8Array(bech32.fromWords(bech32.decode(humanAddress).words));
}
export function humanAddress(
  canonicalAddress: Uint8Array,
  prefix: string = "terra"
) {
  return bech32.encode(prefix, bech32.toWords(canonicalAddress));
}

export const isNativeDenomInjective = (string = "") =>
  string === "inj" || string.startsWith("peggy0x");

export function isNativeCosmWasmDenom(
  chainId: CosmWasmChainId,
  address: string
) {
  return (
    (isTerraChain(chainId) && isNativeDenom(address)) ||
    (chainId === CHAIN_ID_INJECTIVE && isNativeDenomInjective(address))
  );
}

export function buildTokenId(
  // chainId: ChainId,
  chainId: typeof CHAIN_ID_TERRA2 | typeof CHAIN_ID_INJECTIVE,
  address: string
) {
  return (
    (isNativeCosmWasmDenom(chainId, address) ? "01" : "00") +
    keccak256(Buffer.from(address, "utf-8")).substring(4)
  );
}
