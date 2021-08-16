import { Bridge__factory } from "@certusone/wormhole-sdk";
import { Connection, PublicKey } from "@solana/web3.js";
import { ethers } from "ethers";
import {
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_TOKEN_BRIDGE_ADDRESS,
} from "./consts";

/**
 * Returns whether or not an asset address on Ethereum is a wormhole wrapped asset
 * @param provider
 * @param assetAddress
 * @returns
 */
export async function getIsWrappedAssetEth(
  provider: ethers.providers.Web3Provider,
  assetAddress: string
) {
  if (!assetAddress) return false;
  const tokenBridge = Bridge__factory.connect(
    ETH_TOKEN_BRIDGE_ADDRESS,
    provider
  );
  return await tokenBridge.isWrappedAsset(assetAddress);
}

/**
 * Returns whether or not an asset on Solana is a wormhole wrapped asset
 * @param assetAddress
 * @returns
 */
export async function getIsWrappedAssetSol(mintAddress: string) {
  if (!mintAddress) return false;
  const { wrapped_meta_address } = await import(
    "@certusone/wormhole-sdk/lib/solana/token/token_bridge"
  );
  const wrappedMetaAddress = wrapped_meta_address(
    SOL_TOKEN_BRIDGE_ADDRESS,
    new PublicKey(mintAddress).toBytes()
  );
  const wrappedMetaAddressPK = new PublicKey(wrappedMetaAddress);
  // TODO: share connection in context?
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const wrappedMetaAccountInfo = await connection.getAccountInfo(
    wrappedMetaAddressPK
  );
  return !!wrappedMetaAccountInfo;
}
