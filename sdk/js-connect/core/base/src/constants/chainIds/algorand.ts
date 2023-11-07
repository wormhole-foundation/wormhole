import { ChainName, Network } from "../../constants";
import { RoArray, constMap } from "../../utils";

const networkChainAlgorandChainId = [
  ["Mainnet", [["Algorand", "mainnet-v1.0"]]],
  ["Testnet", [["Algorand", "testnet-v1.0"]]],
  ["Devnet", [["Algorand", "sandnet-v1.0"]]],
] as const satisfies RoArray<readonly [Network, RoArray<readonly [ChainName, string]>]>;

export const algorandChainIdToNetworkChain = constMap(networkChainAlgorandChainId, [2, [0, 1]]);

export const algorandNetworkChainToChainId = constMap(networkChainAlgorandChainId);
