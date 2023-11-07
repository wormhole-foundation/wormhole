import { ChainName, Network } from "../../constants";
import { RoArray, constMap } from "../../utils";

const networkChainSolanaGenesisHashes = [
  ["Mainnet", [["Solana", "5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d"]]],
  ["Testnet", [["Solana", "EtWTRABZaYq6iMfeYKouRu166VU2xqa1wcaWoxPkrZBG"]]], // Note: this is referred to as `devnet` in sol
  ["Devnet", [["Solana", ""]]], // Note: this is only for local testing with Tilt
] as const satisfies RoArray<readonly [Network, RoArray<readonly [ChainName, string]>]>;

export const solGenesisHashToNetworkChainPair = constMap(networkChainSolanaGenesisHashes, [
  2,
  [0, 1],
]);

export const solNetworkChainToGenesisHash = constMap(networkChainSolanaGenesisHashes);
