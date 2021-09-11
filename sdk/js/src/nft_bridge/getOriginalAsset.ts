import { Connection, PublicKey } from "@solana/web3.js";
import { BigNumber, ethers } from "ethers";
import { arrayify, zeroPad } from "ethers/lib/utils";
import { TokenImplementation__factory } from "../ethers-contracts";
import { ChainId, CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "../utils";
import { getIsWrappedAssetEth } from "./getIsWrappedAsset";

export interface WormholeWrappedNFTInfo {
  isWrapped: boolean;
  chainId: ChainId;
  assetAddress: Uint8Array;
  tokenId?: string;
}

/**
 * Returns a origin chain and asset address on {originChain} for a provided Wormhole wrapped address
 * @param tokenBridgeAddress
 * @param provider
 * @param wrappedAddress
 * @returns
 */
export async function getOriginalAssetEth(
  tokenBridgeAddress: string,
  provider: ethers.providers.Web3Provider,
  wrappedAddress: string,
  tokenId: string
): Promise<WormholeWrappedNFTInfo> {
  const isWrapped = await getIsWrappedAssetEth(
    tokenBridgeAddress,
    provider,
    wrappedAddress
  );
  if (isWrapped) {
    const token = TokenImplementation__factory.connect(
      wrappedAddress,
      provider
    );
    const chainId = (await token.chainId()) as ChainId; // origin chain
    const assetAddress = await token.nativeContract(); // origin address
    return {
      isWrapped: true,
      chainId,
      assetAddress: arrayify(assetAddress),
      tokenId, // tokenIds are maintained across EVM chains
    };
  }
  return {
    isWrapped: false,
    chainId: CHAIN_ID_ETH,
    assetAddress: zeroPad(arrayify(wrappedAddress), 32),
    tokenId,
  };
}

/**
 * Returns a origin chain and asset address on {originChain} for a provided Wormhole wrapped address
 * @param connection
 * @param tokenBridgeAddress
 * @param mintAddress
 * @returns
 */
export async function getOriginalAssetSol(
  connection: Connection,
  tokenBridgeAddress: string,
  mintAddress: string
): Promise<WormholeWrappedNFTInfo> {
  if (mintAddress) {
    // TODO: share some of this with getIsWrappedAssetSol, like a getWrappedMetaAccountAddress or something
    const { parse_wrapped_meta, wrapped_meta_address } = await import(
      "../solana/nft/nft_bridge"
    );
    const wrappedMetaAddress = wrapped_meta_address(
      tokenBridgeAddress,
      new PublicKey(mintAddress).toBytes()
    );
    const wrappedMetaAddressPK = new PublicKey(wrappedMetaAddress);
    const wrappedMetaAccountInfo = await connection.getAccountInfo(
      wrappedMetaAddressPK
    );
    if (wrappedMetaAccountInfo) {
      const parsed = parse_wrapped_meta(wrappedMetaAccountInfo.data);
      const token_id_arr = parsed.token_id as BigUint64Array;
      const token_id = BigNumber.from(token_id_arr.reverse()).toString();
      return {
        isWrapped: true,
        chainId: parsed.chain,
        assetAddress: parsed.token_address,
        tokenId: token_id,
      };
    }
  }
  try {
    return {
      isWrapped: false,
      chainId: CHAIN_ID_SOLANA,
      assetAddress: new PublicKey(mintAddress).toBytes(),
    };
  } catch (e) {}
  return {
    isWrapped: false,
    chainId: CHAIN_ID_SOLANA,
    assetAddress: new Uint8Array(32),
  };
}
