import { PublicKey } from "@solana/web3.js";
import { ethers } from "ethers";
import { CHAIN_ID_SOLANA } from "..";
import { NFTBridge__factory } from "../ethers-contracts";
import { ChainId } from "../utils";

/**
 * Returns a foreign asset address on Ethereum for a provided native chain and asset address, AddressZero if it does not exist
 * @param tokenBridgeAddress
 * @param provider
 * @param originChain
 * @param originAsset zero pad to 32 bytes
 * @returns
 */
export async function getForeignAssetEth(
  tokenBridgeAddress: string,
  provider: ethers.providers.Web3Provider,
  originChain: ChainId,
  originAsset: Uint8Array
) {
  const tokenBridge = NFTBridge__factory.connect(tokenBridgeAddress, provider);
  try {
    if (originChain === CHAIN_ID_SOLANA) {
      // All NFTs from Solana are minted to the same address, the originAsset is encoded as the tokenId as
      // BigNumber.from(new PublicKey(originAsset).toBytes()).toString()
      const addr = await tokenBridge.wrappedAsset(
        originChain,
        "0x0101010101010101010101010101010101010101010101010101010101010101"
      );
      return addr;
    }
    return await tokenBridge.wrappedAsset(originChain, originAsset);
  } catch (e) {
    return null;
  }
}
/**
 * Returns a foreign asset address on Solana for a provided native chain and asset address
 * @param tokenBridgeAddress
 * @param originChain
 * @param originAsset zero pad to 32 bytes
 * @returns
 */
export async function getForeignAssetSol(
  tokenBridgeAddress: string,
  originChain: ChainId,
  originAsset: Uint8Array,
  tokenId: Uint8Array
) {
  const { wrapped_address } = await import("../solana/nft/nft_bridge");
  const wrappedAddress = wrapped_address(
    tokenBridgeAddress,
    originAsset,
    originChain,
    tokenId
  );
  const wrappedAddressPK = new PublicKey(wrappedAddress);
  // we don't require NFT accounts to exist, so don't check them.
  return wrappedAddressPK.toString();
}
