import { ChainName, Network } from "../../constants";
import { RoArray, constMap } from "../../utils";

const networkChainSuiChainId = [
  ["Mainnet", [["Sui", "35834a8a"]]],
  ["Testnet", [["Sui", "4c78adac"]]],
  ["Devnet", [["Sui", ""]]],
] as const satisfies RoArray<readonly [Network, RoArray<readonly [ChainName, string]>]>;

export const suiChainIdToNetworkChain = constMap(networkChainSuiChainId, [2, [0, 1]]);

export const suiNetworkChainToChainId = constMap(networkChainSuiChainId);
