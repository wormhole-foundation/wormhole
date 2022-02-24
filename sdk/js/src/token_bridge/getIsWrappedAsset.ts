import { Connection, PublicKey } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { ethers } from "ethers";
import { Bridge__factory } from "../ethers-contracts";
import { importTokenWasm } from "../solana/wasm";

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
) {
  if (!assetAddress) return false;
  const tokenBridge = Bridge__factory.connect(tokenBridgeAddress, provider);
  return await tokenBridge.isWrappedAsset(assetAddress);
}

export async function getIsWrappedAssetTerra(
  tokenBridgeAddress: string,
  client: LCDClient,
  assetAddress: string
) {
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
) {
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
