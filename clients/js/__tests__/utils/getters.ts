import { CHAINS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";

export type WormholeSDKChainName = keyof typeof CHAINS;

export const getChains = () => {
  return Object.keys(CHAINS) as WormholeSDKChainName[];
};

export type WormholeSDKNetwork = "mainnet" | "testnet" | "devnet";

export const getNetworks = () => {
  return ["mainnet", "testnet", "devnet"] as WormholeSDKNetwork[];
};
