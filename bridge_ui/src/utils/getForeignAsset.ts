import {
  ChainId,
  CHAIN_ID_SOLANA,
  getForeignAssetEth as getForeignAssetEthTx,
  getForeignAssetSolana as getForeignAssetSolanaTx,
  getForeignAssetTerra as getForeignAssetTerraTx,
} from "@certusone/wormhole-sdk";
import { Connection, PublicKey } from "@solana/web3.js";
import { ethers } from "ethers";
import { arrayify, isHexString, zeroPad } from "ethers/lib/utils";
import { LCDClient } from "@terra-money/terra.js";
import {
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  TERRA_HOST,
  SOL_TOKEN_BRIDGE_ADDRESS,
  TERRA_TOKEN_BRIDGE_ADDRESS,
} from "./consts";

export async function getForeignAssetEth(
  provider: ethers.providers.Web3Provider,
  originChain: ChainId,
  originAsset: string
) {
  try {
    // TODO: address conversion may be more complex than this
    const originAssetBytes = zeroPad(
      originChain === CHAIN_ID_SOLANA
        ? new PublicKey(originAsset).toBytes()
        : arrayify(originAsset),
      32
    );
    return await getForeignAssetEthTx(
      ETH_TOKEN_BRIDGE_ADDRESS,
      provider,
      originChain,
      originAssetBytes
    );
  } catch (e) {
    return ethers.constants.AddressZero;
  }
}

export async function getForeignAssetSol(
  originChain: ChainId,
  originAsset: string
) {
  if (!isHexString(originAsset)) return null;
  // TODO: address conversion may be more complex than this
  const originAssetBytes = zeroPad(
    arrayify(originAsset, { hexPad: "left" }),
    32
  );
  // TODO: share connection in context?
  const connection = new Connection(SOLANA_HOST, "confirmed");
  return await getForeignAssetSolanaTx(
    connection,
    SOL_TOKEN_BRIDGE_ADDRESS,
    originChain,
    originAssetBytes
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
    const originAssetBytes = zeroPad(
      originChain === CHAIN_ID_SOLANA
        ? new PublicKey(originAsset).toBytes()
        : arrayify(originAsset),
      32
    );
    const lcd = new LCDClient(TERRA_HOST);
    return await getForeignAssetTerraTx(
      TERRA_TOKEN_BRIDGE_ADDRESS,
      lcd,
      originChain,
      originAssetBytes
    );
  } catch (e) {
    return ethers.constants.AddressZero;
  }
}
