import { Connection, PublicKey } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { Algodv2, getApplicationAddress } from "algosdk";
import { ethers } from "ethers";
import { Bridge__factory } from "../ethers-contracts";
import { importTokenWasm } from "../solana/wasm";
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
