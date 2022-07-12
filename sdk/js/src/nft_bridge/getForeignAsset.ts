import { BN } from "@project-serum/anchor";
import { PublicKeyInitData } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { ethers } from "ethers";
import { isBytes } from "ethers/lib/utils";
import { fromUint8Array } from "js-base64";
import { CHAIN_ID_SOLANA } from "..";
import { NFTBridge__factory } from "../ethers-contracts";
import { deriveWrappedMintKey } from "../solana/nftBridge";
import { ChainId, ChainName, coalesceChainId } from "../utils";

/**
 * Returns a foreign asset address on Ethereum for a provided native chain and asset address, AddressZero if it does not exist
 * @param nftBridgeAddress
 * @param provider
 * @param originChain
 * @param originAsset zero pad to 32 bytes
 * @returns
 */
export async function getForeignAssetEth(
  nftBridgeAddress: string,
  provider: ethers.Signer | ethers.providers.Provider,
  originChain: ChainId | ChainName,
  originAsset: Uint8Array
): Promise<string | null> {
  const originChainId = coalesceChainId(originChain);
  const tokenBridge = NFTBridge__factory.connect(nftBridgeAddress, provider);
  try {
    if (originChainId === CHAIN_ID_SOLANA) {
      // All NFTs from Solana are minted to the same address, the originAsset is encoded as the tokenId as
      // BigNumber.from(new PublicKey(originAsset).toBytes()).toString()
      const addr = await tokenBridge.wrappedAsset(
        originChain,
        "0x0101010101010101010101010101010101010101010101010101010101010101"
      );
      return addr;
    }
    return await tokenBridge.wrappedAsset(originChainId, originAsset);
  } catch (e) {
    return null;
  }
}

/**
 * Returns a foreign asset address on Terra for a provided native chain and asset address
 * @param nftBridgeAddress
 * @param client
 * @param originChain
 * @param originAsset
 * @returns
 */
export async function getForeignAssetTerra(
  nftBridgeAddress: string,
  client: LCDClient,
  originChain: ChainId,
  originAsset: Uint8Array
): Promise<string | null> {
  const originChainId = coalesceChainId(originChain);
  try {
    const address =
      originChain == CHAIN_ID_SOLANA
        ? "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
        : fromUint8Array(originAsset);
    const result: { address: string } = await client.wasm.contractQuery(
      nftBridgeAddress,
      {
        wrapped_registry: {
          chain: originChainId,
          address,
        },
      }
    );
    return result.address;
  } catch (e) {
    return null;
  }
}

/**
 * Returns a foreign asset address on Solana for a provided native chain and asset address
 * @param nftBridgeAddress
 * @param originChain
 * @param originAsset zero pad to 32 bytes
 * @returns
 */
export async function getForeignAssetSolana(
  nftBridgeAddress: PublicKeyInitData,
  originChain: ChainId | ChainName,
  originAsset: string | Uint8Array | Buffer,
  tokenId: Uint8Array | Buffer | bigint
): Promise<string> {
  // we don't require NFT accounts to exist, so don't check them.
  return deriveWrappedMintKey(
    nftBridgeAddress,
    coalesceChainId(originChain) as number,
    originAsset,
    isBytes(tokenId) ? BigInt(new BN(tokenId).toString()) : tokenId
  ).toString();
}

export const getForeignAssetSol = getForeignAssetSolana;
