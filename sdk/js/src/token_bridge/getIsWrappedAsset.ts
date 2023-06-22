import { JsonRpcProvider } from "@mysten/sui.js";
import { Commitment, Connection, PublicKeyInitData } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { Algodv2, getApplicationAddress } from "algosdk";
import { AptosClient } from "aptos";
import { ethers } from "ethers";
import { Bridge__factory } from "../ethers-contracts";
import { getWrappedMeta } from "../solana/tokenBridge";
import { getTokenFromTokenRegistry } from "../sui";
import { coalesceModuleAddress, ensureHexPrefix } from "../utils";
import { safeBigIntToNumber } from "../utils/bigint";

/**
 * Returns whether or not an asset address on Ethereum is a wormhole wrapped asset
 * @param tokenBridgeAddress
 * @param provider
 * @param assetAddress
 * @returns
 */
export async function getIsWrappedAssetEth(
  tokenBridgeAddress: string,
  provider: ethers.Signer | ethers.providers.Provider,
  assetAddress: string
): Promise<boolean> {
  if (!assetAddress) return false;
  const tokenBridge = Bridge__factory.connect(tokenBridgeAddress, provider);
  return await tokenBridge.isWrappedAsset(assetAddress);
}

// TODO: this doesn't seem right
export async function getIsWrappedAssetTerra(
  tokenBridgeAddress: string,
  client: LCDClient,
  assetAddress: string
): Promise<boolean> {
  return false;
}

/**
 * Returns whether or not an asset on Solana is a wormhole wrapped asset
 * @param connection
 * @param tokenBridgeAddress
 * @param mintAddress
 * @param [commitment]
 * @returns
 */
export async function getIsWrappedAssetSolana(
  connection: Connection,
  tokenBridgeAddress: PublicKeyInitData,
  mintAddress: PublicKeyInitData,
  commitment?: Commitment
): Promise<boolean> {
  if (!mintAddress) {
    return false;
  }
  return getWrappedMeta(connection, tokenBridgeAddress, mintAddress, commitment)
    .catch((_) => null)
    .then((meta) => meta != null);
}

export const getIsWrappedAssetSol = getIsWrappedAssetSolana;

/**
 * Returns whethor or not an asset on Algorand is a wormhole wrapped asset
 * @param client Algodv2 client
 * @param tokenBridgeId token bridge ID
 * @param assetId Algorand asset index
 * @returns true if the asset is wrapped
 */
export async function getIsWrappedAssetAlgorand(
  client: Algodv2,
  tokenBridgeId: bigint,
  assetId: bigint
): Promise<boolean> {
  if (assetId === BigInt(0)) {
    return false;
  }
  const tbAddr: string = getApplicationAddress(tokenBridgeId);
  const assetInfo = await client.getAssetByID(safeBigIntToNumber(assetId)).do();
  const creatorAddr = assetInfo.params.creator;
  const creatorAcctInfo = await client.accountInformation(creatorAddr).do();
  const wormhole: boolean = creatorAcctInfo["auth-addr"] === tbAddr;
  return wormhole;
}

export function getIsWrappedAssetNear(
  tokenBridge: string,
  asset: string
): boolean {
  return asset.endsWith("." + tokenBridge);
}

/**
 * Determines whether or not given address is wrapped or native to Aptos.
 * @param client Client used to transfer data to/from Aptos node
 * @param tokenBridgeAddress Address of token bridge
 * @param assetFullyQualifiedType Fully qualified type of asset
 * @returns True if asset is wrapped
 */
export async function getIsWrappedAssetAptos(
  client: AptosClient,
  tokenBridgeAddress: string,
  assetFullyQualifiedType: string
): Promise<boolean> {
  assetFullyQualifiedType = ensureHexPrefix(assetFullyQualifiedType);
  try {
    // get origin info from asset address
    await client.getAccountResource(
      coalesceModuleAddress(assetFullyQualifiedType),
      `${tokenBridgeAddress}::state::OriginInfo`
    );
    return true;
  } catch {
    return false;
  }
}

export async function getIsWrappedAssetSui(
  provider: JsonRpcProvider,
  tokenBridgeStateObjectId: string,
  type: string
): Promise<boolean> {
  // An easy way to determine if given asset isn't a wrapped asset is to ensure
  // module name and struct name are coin and COIN respectively.
  if (!type.endsWith("::coin::COIN")) {
    return false;
  }

  const response = await getTokenFromTokenRegistry(
    provider,
    tokenBridgeStateObjectId,
    type
  );
  if (!response.error) {
    return response.data?.type?.includes("WrappedAsset") || false;
  }

  if (response.error.code === "dynamicFieldNotFound") {
    return false;
  }

  throw new Error(
    `Unexpected getDynamicFieldObject response ${response.error}`
  );
}
