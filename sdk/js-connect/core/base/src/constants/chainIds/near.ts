import { ChainName, Network } from "../../constants";
import { RoArray, constMap } from "../../utils";

const networkChainNearChainId = [
  ["Mainnet", [["Near", "mainnet"]]],
  ["Testnet", [["Near", "testnet"]]],
  ["Devnet", [["Near", ""]]],
] as const satisfies RoArray<readonly [Network, RoArray<readonly [ChainName, string]>]>;

export const nearChainIdToNetworkChain = constMap(networkChainNearChainId, [2, [0, 1]]);

export const nearNetworkChainToChainId = constMap(networkChainNearChainId);
