import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  TokenImplementation__factory,
} from "@certusone/wormhole-sdk";
import { Connection, PublicKey } from "@solana/web3.js";
import { ethers } from "ethers";
import { arrayify } from "ethers/lib/utils";
import { uint8ArrayToHex } from "./array";
import { SOLANA_HOST, SOL_TOKEN_BRIDGE_ADDRESS } from "./consts";
import { getIsWrappedAssetEth } from "./getIsWrappedAsset";

export interface WormholeWrappedInfo {
  isWrapped: boolean;
  chainId: ChainId;
  assetAddress: string;
}

/**
 * Returns a origin chain and asset address on {originChain} for a provided Wormhole wrapped address
 * @param provider
 * @param wrappedAddress
 * @returns
 */
export async function getOriginalAssetEth(
  provider: ethers.providers.Web3Provider,
  wrappedAddress: string
): Promise<WormholeWrappedInfo> {
  const isWrapped = await getIsWrappedAssetEth(provider, wrappedAddress);
  if (isWrapped) {
    const token = TokenImplementation__factory.connect(
      wrappedAddress,
      provider
    );
    const chainId = (await token.chainId()) as ChainId; // origin chain
    const assetAddress = await token.nativeContract(); // origin address
    // TODO: type this?
    return {
      isWrapped: true,
      chainId,
      assetAddress: uint8ArrayToHex(arrayify(assetAddress)),
    };
  }
  return {
    isWrapped: false,
    chainId: CHAIN_ID_ETH,
    assetAddress: wrappedAddress,
  };
}

export async function getOriginalAssetSol(
  mintAddress: string
): Promise<WormholeWrappedInfo> {
  if (mintAddress) {
    // TODO: share some of this with getIsWrappedAssetSol, like a getWrappedMetaAccountAddress or something
    const { parse_wrapped_meta, wrapped_meta_address } = await import(
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
    if (wrappedMetaAccountInfo) {
      const parsed = parse_wrapped_meta(wrappedMetaAccountInfo.data);
      return {
        isWrapped: true,
        chainId: parsed.chain,
        assetAddress: uint8ArrayToHex(parsed.token_address),
      };
    }
  }
  return {
    isWrapped: false,
    chainId: CHAIN_ID_SOLANA,
    assetAddress: mintAddress,
  };
}
