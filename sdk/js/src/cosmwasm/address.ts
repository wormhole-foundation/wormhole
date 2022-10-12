import { LCDClient as TerraLCDClient } from "@terra-money/terra.js";
import { LCDClient as XplaLCDClient } from "@xpla/xpla.js";
import { keccak256 } from "ethers/lib/utils";
import { isNativeDenom } from "../terra";
import {
  CHAIN_ID_INJECTIVE,
  CHAIN_ID_TERRA,
  CHAIN_ID_XPLA,
  coalesceCosmWasmChainId,
  CosmWasmChainId,
  CosmWasmChainName,
  isTerraChain,
} from "../utils";

export const isNativeDenomInjective = (denom: string) => denom === "inj";
export const isNativeDenomXpla = (denom: string) => denom === "axpla";

export function isNativeCosmWasmDenom(
  chainId: CosmWasmChainId,
  address: string
) {
  return (
    (isTerraChain(chainId) && isNativeDenom(address)) ||
    (chainId === CHAIN_ID_INJECTIVE && isNativeDenomInjective(address)) ||
    (chainId === CHAIN_ID_XPLA && isNativeDenomXpla(address))
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
export interface ExternalIdResponse {
  token_id: {
    Bank?: { denom: string };
    Contract?: {
      NativeCW20?: {
        contract_address: string;
      };
      ForeignToken?: {
        chain_id: string;
        foreign_address: string;
      };
    };
  };
}

// returns the TokenId corresponding to the ExternalTokenId
// see cosmwasm token_addresses.rs
export const queryExternalId = async (
  client: TerraLCDClient | XplaLCDClient,
  tokenBridgeAddress: string,
  externalTokenId: string
) => {
  try {
    const response = await client.wasm.contractQuery<ExternalIdResponse>(
      tokenBridgeAddress,
      {
        external_id: {
          external_id: Buffer.from(externalTokenId, "hex").toString("base64"),
        },
      }
    );
    return (
      // response depends on the token type
      response.token_id.Bank?.denom ||
      response.token_id.Contract?.NativeCW20?.contract_address ||
      response.token_id.Contract?.ForeignToken?.foreign_address
    );
  } catch {
    return null;
  }
};
