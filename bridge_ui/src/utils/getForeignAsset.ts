import {
  ChainId,
  getForeignAssetEth as getForeignAssetEthTx,
  getForeignAssetSolana as getForeignAssetSolanaTx,
  getForeignAssetTerra as getForeignAssetTerraTx,
} from "@certusone/wormhole-sdk";
import { Connection } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { ethers } from "ethers";
import { hexToUint8Array } from "./array";
import {
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_TOKEN_BRIDGE_ADDRESS,
  TERRA_HOST,
  TERRA_TOKEN_BRIDGE_ADDRESS,
} from "./consts";

export async function getForeignAssetEth(
  provider: ethers.providers.Web3Provider,
  originChain: ChainId,
  originAsset: string
) {
  try {
    return await getForeignAssetEthTx(
      ETH_TOKEN_BRIDGE_ADDRESS,
      provider,
      originChain,
      hexToUint8Array(originAsset)
    );
  } catch (e) {
    return null;
  }
}

export async function getForeignAssetSol(
  originChain: ChainId,
  originAsset: string
) {
  const connection = new Connection(SOLANA_HOST, "confirmed");
  return await getForeignAssetSolanaTx(
    connection,
    SOL_TOKEN_BRIDGE_ADDRESS,
    originChain,
    hexToUint8Array(originAsset)
  );
}

/**
 * Returns a foreign asset address on Terra for a provided native chain and asset address
 * @param originChain
 * @param originAsset
 * @returns
 */
export async function getForeignAssetTerra(
  originChain: ChainId,
  originAsset: string
) {
  try {
    const lcd = new LCDClient(TERRA_HOST);
    return await getForeignAssetTerraTx(
      TERRA_TOKEN_BRIDGE_ADDRESS,
      lcd,
      originChain,
      hexToUint8Array(originAsset)
    );
  } catch (e) {
    return null;
  }
}
