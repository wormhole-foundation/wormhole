import { Commitment, Connection, PublicKeyInitData } from "@solana/web3.js";
import { ethers } from "ethers";
import { Bridge__factory } from "../ethers-contracts";
import { getWrappedMeta } from "../solana/nftBridge";

/**
 * Returns whether or not an asset address on Ethereum is a wormhole wrapped asset
 * @param nftBridgeAddress
 * @param provider
 * @param assetAddress
 * @returns
 */
export async function getIsWrappedAssetEth(
  nftBridgeAddress: string,
  provider: ethers.Signer | ethers.providers.Provider,
  assetAddress: string
) {
  if (!assetAddress) return false;
  const tokenBridge = Bridge__factory.connect(nftBridgeAddress, provider);
  return await tokenBridge.isWrappedAsset(assetAddress);
}

/**
 * Returns whether or not an asset on Solana is a wormhole wrapped asset
 * @param connection
 * @param nftBridgeAddress
 * @param mintAddress
 * @param [commitment]
 * @returns
 */
export async function getIsWrappedAssetSolana(
  connection: Connection,
  nftBridgeAddress: PublicKeyInitData,
  mintAddress: PublicKeyInitData,
  commitment?: Commitment
) {
  if (!mintAddress) {
    return false;
  }
  return getWrappedMeta(connection, nftBridgeAddress, mintAddress, commitment)
    .catch((_) => null)
    .then((meta) => meta != null);
}

export const getIsWrappedAssetSol = getIsWrappedAssetSolana;
