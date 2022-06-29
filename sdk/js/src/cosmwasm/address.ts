import { keccak256 } from "ethers/lib/utils";
import { isNativeDenom } from "../terra";
import {
  CHAIN_ID_INJECTIVE,
  CHAIN_ID_TERRA,
  coalesceCosmWasmChainId,
  CosmWasmChainId,
  CosmWasmChainName,
  isTerraChain,
} from "../utils";

export const isNativeDenomInjective = (string = "") => string === "inj";

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
  chain: Exclude<
    CosmWasmChainId | CosmWasmChainName,
    typeof CHAIN_ID_TERRA | "terra"
  >,
  address: string
) {
  const chainId: CosmWasmChainId = coalesceCosmWasmChainId(chain);
  return (
    (isNativeCosmWasmDenom(chainId, address) ? "01" : "00") +
    keccak256(Buffer.from(address, "utf-8")).substring(4)
  );
}
