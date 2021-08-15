import { ChainId, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { Connection, PublicKey } from "@solana/web3.js";
import { ethers } from "ethers";
import { arrayify, isHexString, zeroPad } from "ethers/lib/utils";
import { Bridge__factory } from "../ethers-contracts";
import {
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_TOKEN_BRIDGE_ADDRESS,
} from "./consts";

/**
 * Returns a foreign asset address on Ethereum for a provided native chain and asset address
 * @param provider
 * @param originChain
 * @param originAsset
 * @returns
 */
export async function getForeignAssetEth(
  provider: ethers.providers.Web3Provider,
  originChain: ChainId,
  originAsset: string
) {
  const tokenBridge = Bridge__factory.connect(
    ETH_TOKEN_BRIDGE_ADDRESS,
    provider
  );
  try {
    // TODO: address conversion may be more complex than this
    const originAssetBytes = zeroPad(
      originChain === CHAIN_ID_SOLANA
        ? new PublicKey(originAsset).toBytes()
        : arrayify(originAsset),
      32
    );
    return await tokenBridge.wrappedAsset(originChain, originAssetBytes);
  } catch (e) {
    return ethers.constants.AddressZero;
  }
}

/**
 * Returns a foreign asset address on Solana for a provided native chain and asset address
 * @param originChain
 * @param originAsset
 * @returns
 */
export async function getForeignAssetSol(
  originChain: ChainId,
  originAsset: string
) {
  if (!isHexString(originAsset)) return null;
  const { wrapped_address } = await import("token-bridge");
  // TODO: address conversion may be more complex than this
  const originAssetBytes = zeroPad(
    arrayify(originAsset, { hexPad: "left" }),
    32
  );
  const wrappedAddress = wrapped_address(
    SOL_TOKEN_BRIDGE_ADDRESS,
    originAssetBytes,
    originChain
  );
  const wrappedAddressPK = new PublicKey(wrappedAddress);
  // TODO: share connection in context?
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const wrappedAssetAccountInfo = await connection.getAccountInfo(
    wrappedAddressPK
  );
  console.log("WAAI", wrappedAssetAccountInfo);
  return wrappedAssetAccountInfo ? wrappedAddressPK.toString() : null;
}
