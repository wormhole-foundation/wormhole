import { CHAINS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";

export type WormholeSDKChainName = keyof typeof CHAINS;

export const getChains = () => {
  return Object.keys(CHAINS) as WormholeSDKChainName[];
};

export const networks = ["mainnet", "testnet", "devnet"];
