import { CHAINS, Network } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { NETWORKS as RPC_NETWORKS } from "../../src/consts/networks";

export type WormholeSDKChainName = keyof typeof CHAINS;

export const getChains = () => {
  return Object.keys(CHAINS) as WormholeSDKChainName[];
};

export type WormholeSDKNetwork = "mainnet" | "testnet" | "devnet";

export const getNetworks = () => {
  return ["mainnet", "testnet", "devnet"] as WormholeSDKNetwork[];
};

export const getRpcEndpoint = (
  chain: WormholeSDKChainName,
  network: Network
) => {
  return String(RPC_NETWORKS[network][chain].rpc);
};
