// TODO(aki): move to SDK
import {
  buildTokenId,
  isNativeCosmWasmDenom,
} from "@certusone/wormhole-sdk/lib/esm/cosmwasm/address";
import { WormholeWrappedInfo } from "@certusone/wormhole-sdk/lib/esm/token_bridge/getOriginalAsset";
import { hexToUint8Array } from "@certusone/wormhole-sdk/lib/esm/utils/array";
import {
  CHAIN_ID_SEI,
  ChainId,
  ChainName,
  coalesceChainId,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { CosmWasmClient } from "@cosmjs/cosmwasm-stargate";
import { fromUint8Array } from "js-base64";

/**
 * Returns the address of the foreign asset
 * @param tokenBridgeAddress Address of token bridge contact
 * @param client Holds the wallet and signing information
 * @param originChain The chainId of the origin of the asset
 * @param originAsset The address of the origin asset
 * @returns The foreign asset address or null
 */
export async function getForeignAssetSei(
  tokenBridgeAddress: string,
  cosmwasmClient: CosmWasmClient,
  originChain: ChainId | ChainName,
  originAsset: Uint8Array
): Promise<string | null> {
  try {
    const queryResult = await cosmwasmClient.queryContractSmart(
      tokenBridgeAddress,
      {
        wrapped_registry: {
          chain: coalesceChainId(originChain),
          address: fromUint8Array(originAsset),
        },
      }
    );
    return queryResult.address;
  } catch (e) {
    return null;
  }
}

/**
 * Return if the VAA has been redeemed or not
 * @param tokenBridgeAddress The Sei token bridge contract address
 * @param signedVAA The signed VAA byte array
 * @param client Holds the wallet and signing information
 * @returns true if the VAA has been redeemed.
 */
export async function getIsTransferCompletedSei(
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
  client: CosmWasmClient
): Promise<boolean> {
  const queryResult = await client.queryContractSmart(tokenBridgeAddress, {
    is_vaa_redeemed: {
      vaa: fromUint8Array(signedVAA),
    },
  });
  return queryResult.is_redeemed;
}

/**
 * Returns information about the asset
 * @param wrappedAddress Address of the asset in wormhole wrapped format (hex string)
 * @param client WASM api client
 * @returns Information about the asset
 */
export async function getOriginalAssetSei(
  wrappedAddress: string,
  client: CosmWasmClient
): Promise<WormholeWrappedInfo> {
  const chainId = CHAIN_ID_SEI;
  if (isNativeCosmWasmDenom(chainId, wrappedAddress)) {
    return {
      isWrapped: false,
      chainId,
      assetAddress: hexToUint8Array(buildTokenId(chainId, wrappedAddress)),
    };
  }
  try {
    const response = await client.queryContractSmart(wrappedAddress, {
      wrapped_asset_info: {},
    });
    return {
      isWrapped: true,
      chainId: response.asset_chain,
      assetAddress: new Uint8Array(
        Buffer.from(response.asset_address, "base64")
      ),
    };
  } catch {}
  return {
    isWrapped: false,
    chainId: chainId,
    assetAddress: hexToUint8Array(buildTokenId(chainId, wrappedAddress)),
  };
}

export const queryExternalIdSei = async (
  client: CosmWasmClient,
  tokenBridgeAddress: string,
  externalTokenId: string
): Promise<string | null> => {
  try {
    const response = await client.queryContractSmart(tokenBridgeAddress, {
      external_id: {
        external_id: Buffer.from(externalTokenId, "hex").toString("base64"),
      },
    });
    const denomOrAddress: string | undefined =
      response.token_id.Bank?.denom ||
      response.token_id.Contract?.NativeCW20?.contract_address ||
      response.token_id.Contract?.ForeignToken?.foreign_address;
    return denomOrAddress || null;
  } catch {
    return null;
  }
};

/**
 * Submits the supplied VAA to Sei
 * @param signedVAA VAA with the attestation message
 * @returns Message to be broadcast
 */
export async function submitVAAOnSei(signedVAA: Uint8Array) {
  return {
    submit_vaa: { data: fromUint8Array(signedVAA) },
  };
}

export const createWrappedOnSei = submitVAAOnSei;
export const updateWrappedOnSei = submitVAAOnSei;
export const redeemOnSei = submitVAAOnSei;

export function parseSequenceFromLogSei(info: any): string {
  // Scan for the Sequence attribute in all the outputs of the transaction.
  let sequence = "";
  info.logs.forEach((row: any) => {
    row.events.forEach((event: any) => {
      event.attributes.forEach((attribute: any) => {
        if (attribute.key === "message.sequence") {
          sequence = attribute.value;
        }
      });
    });
  });
  return sequence.toString();
}
