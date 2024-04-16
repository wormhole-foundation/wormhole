import { LCDClient as TerraLCDClient } from "@terra-money/terra.js";
import { LCDClient as XplaLCDClient } from "@xpla/xpla.js";
import { keccak256 } from "ethers/lib/utils";
import { isNativeDenom } from "../terra";
import {
  CHAIN_ID_COSMOSHUB,
  CHAIN_ID_EVMOS,
  CHAIN_ID_WORMCHAIN,
  CHAIN_ID_INJECTIVE,
  CHAIN_ID_SEI,
  CHAIN_ID_TERRA,
  CHAIN_ID_XPLA,
  coalesceCosmWasmChainId,
  CosmWasmChainId,
  CosmWasmChainName,
  isTerraChain,
  CHAIN_ID_KUJIRA,
  CHAIN_ID_NEUTRON,
  CHAIN_ID_OSMOSIS,
  CHAIN_ID_CELESTIA,
  CHAIN_ID_STARGAZE,
  CHAIN_ID_SEDA,
  CHAIN_ID_DYMENSION,
  CHAIN_ID_PROVENANCE,
} from "../utils";

export const isNativeDenomInjective = (denom: string) => denom === "inj";
export const isNativeDenomXpla = (denom: string) => denom === "axpla";
export const isNativeDenomSei = (denom: string) => denom === "usei";
export const isNativeDenomWormchain = (denom: string) => denom === "uworm";
export const isNativeDenomOsmosis = (denom: string) => denom === "uosmo";
export const isNativeDenomCosmosHub = (denom: string) => denom === "uatom";
export const isNativeDenomEvmos = (denom: string) =>
  denom === "aevmos" || denom === "atevmos";
export const isNativeDenomKujira = (denom: string) => denom === "ukuji";
export const isNativeDenomNeutron = (denom: string) => denom === "untrn";
export const isNativeDenomCelestia = (denom: string) => denom === "utia";
export const isNativeDenomStargaze = (denom: string) => denom === "ustars";
export const isNativeDenomSeda = (denom: string) => denom === "aseda";
export const isNativeDenomDymension = (denom: string) => denom === "adym";
export const isNativeDenomProvenance = (denom: string) => denom === "nhash";

export function isNativeCosmWasmDenom(
  chainId: CosmWasmChainId,
  address: string
) {
  return (
    (isTerraChain(chainId) && isNativeDenom(address)) ||
    (chainId === CHAIN_ID_INJECTIVE && isNativeDenomInjective(address)) ||
    (chainId === CHAIN_ID_XPLA && isNativeDenomXpla(address)) ||
    (chainId === CHAIN_ID_SEI && isNativeDenomSei(address)) ||
    (chainId === CHAIN_ID_WORMCHAIN && isNativeDenomWormchain(address)) ||
    (chainId === CHAIN_ID_OSMOSIS && isNativeDenomOsmosis(address)) ||
    (chainId === CHAIN_ID_COSMOSHUB && isNativeDenomCosmosHub(address)) ||
    (chainId === CHAIN_ID_EVMOS && isNativeDenomEvmos(address)) ||
    (chainId === CHAIN_ID_KUJIRA && isNativeDenomKujira(address)) ||
    (chainId === CHAIN_ID_NEUTRON && isNativeDenomNeutron(address)) ||
    (chainId === CHAIN_ID_CELESTIA && isNativeDenomCelestia(address)) ||
    (chainId === CHAIN_ID_STARGAZE && isNativeDenomStargaze(address)) ||
    (chainId === CHAIN_ID_SEDA && isNativeDenomSeda(address)) ||
    (chainId === CHAIN_ID_DYMENSION && isNativeDenomDymension(address)) ||
    (chainId === CHAIN_ID_PROVENANCE && isNativeDenomProvenance(address))
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
