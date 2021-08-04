import { PublicKey } from "@solana/web3.js";
import { ethers } from "ethers";
import { arrayify, zeroPad } from "ethers/lib/utils";
import { Bridge__factory } from "../ethers-contracts";
import { ChainId, CHAIN_ID_SOLANA, ETH_TOKEN_BRIDGE_ADDRESS } from "./consts";

export function wrappedAssetEth(
  provider: ethers.providers.Web3Provider,
  originChain: ChainId,
  originAsset: string
) {
  const tokenBridge = Bridge__factory.connect(
    ETH_TOKEN_BRIDGE_ADDRESS,
    provider
  );
  // TODO: address conversion may be more complex than this
  const originAssetBytes = zeroPad(
    originChain === CHAIN_ID_SOLANA
      ? new PublicKey(originAsset).toBytes()
      : arrayify(originAsset),
    32
  );
  return tokenBridge.wrappedAsset(originChain, originAssetBytes);
}
