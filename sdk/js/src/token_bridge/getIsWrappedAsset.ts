import { ChainGrpcWasmApi } from "@injectivelabs/sdk-ts";
import { Connection, PublicKey } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { Algodv2, getApplicationAddress } from "algosdk";
import { AptosClient } from "aptos";
import { ethers } from "ethers";
import { Bridge__factory } from "../ethers-contracts";
import { importTokenWasm } from "../solana/wasm";
import { CHAIN_ID_INJECTIVE, ensureHexPrefix, tryNativeToHexString } from "../utils";
import { safeBigIntToNumber } from "../utils/bigint";
import { getForeignAssetInjective } from "./getForeignAsset";

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
 * Checks if the asset is a wrapped asset
 * @param tokenBridgeAddress The address of the Injective token bridge contract
 * @param client Connection/wallet information
 * @param assetAddress Address of the asset in Injective format
 * @returns true if asset is a wormhole wrapped asset
 */
export async function getIsWrappedAssetInjective(
  tokenBridgeAddress: string,
  client: ChainGrpcWasmApi,
  assetAddress: string
): Promise<boolean> {
  const hexified = tryNativeToHexString(assetAddress, "injective");
  const result = await getForeignAssetInjective(
    tokenBridgeAddress,
    client,
    CHAIN_ID_INJECTIVE,
    new Uint8Array(Buffer.from(hexified))
  );
  if (result === null) {
    return false;
  }
  return true;
}

/**
 * Returns whether or not an asset on Solana is a wormhole wrapped asset
 * @param connection
 * @param tokenBridgeAddress
 * @param mintAddress
 * @returns
 */
export async function getIsWrappedAssetSol(
  connection: Connection,
  tokenBridgeAddress: string,
  mintAddress: string
): Promise<boolean> {
  if (!mintAddress) return false;
  const { wrapped_meta_address } = await importTokenWasm();
  const wrappedMetaAddress = wrapped_meta_address(
    tokenBridgeAddress,
    new PublicKey(mintAddress).toBytes()
  );
  const wrappedMetaAddressPK = new PublicKey(wrappedMetaAddress);
  const wrappedMetaAccountInfo = await connection.getAccountInfo(
    wrappedMetaAddressPK
  );
  return !!wrappedMetaAccountInfo;
}

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

// TODO: do we need to check if token is registered in bridge?
export async function getIsWrappedAssetAptos(
  client: AptosClient,
  tokenBridgeAddress: string,
  assetAddress: string,
): Promise<boolean> {
  assetAddress = ensureHexPrefix(assetAddress);
  try {
    // get origin info from asset address
    await client.getAccountResource(assetAddress, `${tokenBridgeAddress}::state::OriginInfo`);
    return true;
  } catch {
    return false;
  }
}
