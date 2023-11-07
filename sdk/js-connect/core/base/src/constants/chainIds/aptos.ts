import { ChainName, Network } from "../../constants";
import { RoArray, constMap } from "../../utils";

const networkChainAptosChainId = [
  ["Mainnet", [["Aptos", 1n]]],
  ["Testnet", [["Aptos", 2n]]],
  ["Devnet", [["Aptos", 0n]]],
] as const satisfies RoArray<readonly [Network, RoArray<readonly [ChainName, bigint]>]>;

export const aptosChainIdToNetworkChain = constMap(networkChainAptosChainId, [2, [0, 1]]);

export const aptosNetworkChainToChainId = constMap(networkChainAptosChainId);
