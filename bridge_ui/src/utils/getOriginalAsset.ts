import {
  ChainId,
  getOriginalAssetEth as getOriginalAssetEthTx,
  getOriginalAssetSol as getOriginalAssetSolTx,
  WormholeWrappedInfo,
} from "@certusone/wormhole-sdk";
import { Connection } from "@solana/web3.js";
import { ethers } from "ethers";
import { uint8ArrayToHex } from "./array";
import {
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_TOKEN_BRIDGE_ADDRESS,
  TERRA_TEST_TOKEN_ADDRESS,
} from "./consts";

export interface StateSafeWormholeWrappedInfo {
  isWrapped: boolean;
  chainId: ChainId;
  assetAddress: string;
}

const makeStateSafe = (
  info: WormholeWrappedInfo
): StateSafeWormholeWrappedInfo => ({
  ...info,
  assetAddress: uint8ArrayToHex(info.assetAddress),
});

export async function getOriginalAssetEth(
  provider: ethers.providers.Web3Provider,
  wrappedAddress: string
): Promise<StateSafeWormholeWrappedInfo> {
  return makeStateSafe(
    await getOriginalAssetEthTx(
      ETH_TOKEN_BRIDGE_ADDRESS,
      provider,
      wrappedAddress
    )
  );
}

export async function getOriginalAssetSol(
  mintAddress: string
): Promise<StateSafeWormholeWrappedInfo> {
  // TODO: share connection in context?
  const connection = new Connection(SOLANA_HOST, "confirmed");
  return makeStateSafe(
    await getOriginalAssetSolTx(
      connection,
      SOL_TOKEN_BRIDGE_ADDRESS,
      mintAddress
    )
  );
}

export async function getOriginalAssetTerra(
  mintAddress: string
): Promise<StateSafeWormholeWrappedInfo> {
  return {
    assetAddress: TERRA_TEST_TOKEN_ADDRESS,
    chainId: 3,
    isWrapped: false,
  };
}
