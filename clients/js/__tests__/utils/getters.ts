import { CHAINS, Network } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { NETWORKS as RPC_NETWORKS } from "../../src/consts/networks";

export type WormholeSDKChainName = keyof typeof CHAINS;

export const getChains = () => {
  return Object.keys(CHAINS) as WormholeSDKChainName[];
};

export const networks = ["mainnet", "testnet", "devnet"];

export const getRpcEndpoint = (
  chain: WormholeSDKChainName,
  network: Network
) => {
  return String(RPC_NETWORKS[network][chain].rpc);
};
