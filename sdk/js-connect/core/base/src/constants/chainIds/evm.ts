import { Network, PlatformToChains } from "../../constants";
import { RoArray, constMap } from "../../utils";

const networkChainEvmCIdEntries = [
  [
    "Mainnet",
    [
      ["Acala", 787n],
      ["Arbitrum", 42161n],
      ["Aurora", 1313161554n],
      ["Avalanche", 43114n],
      ["Base", 8453n],
      ["Bsc", 56n],
      ["Celo", 42220n],
      ["Ethereum", 1n],
      ["Fantom", 250n],
      ["Gnosis", 100n],
      ["Karura", 686n],
      ["Klaytn", 8217n],
      ["Moonbeam", 1284n],
      ["Neon", 245022934n],
      ["Oasis", 42262n],
      ["Optimism", 10n],
      ["Polygon", 137n],
      ["Rootstock", 30n],
      ["Sepolia", 0n], // Note: this is a lie but sepolia is just a testnet
    ],
  ],
  [
    "Testnet",
    [
      ["Acala", 597n],
      ["Arbitrum", 421613n], //arbitrum goerli
      ["Aurora", 1313161555n],
      ["Avalanche", 43113n], //fuji
      ["Base", 84531n],
      ["Bsc", 97n],
      ["Celo", 44787n], //alfajores
      ["Ethereum", 5n], //goerli
      ["Fantom", 4002n],
      ["Gnosis", 10200n],
      ["Karura", 596n],
      ["Klaytn", 1001n], //baobab
      ["Moonbeam", 1287n], //moonbase alpha
      ["Neon", 245022940n],
      ["Oasis", 42261n],
      ["Optimism", 420n],
      ["Polygon", 80001n], //mumbai
      ["Rootstock", 31n],
      ["Sepolia", 11155111n], //actually just another ethereum testnet...
    ],
  ],
  ["Devnet", [["Bsc", 1397n]]],
] as const satisfies RoArray<
  readonly [Network, RoArray<readonly [PlatformToChains<"Evm">, bigint]>]
>;

export const evmChainIdToNetworkChainPair = constMap(networkChainEvmCIdEntries, [2, [0, 1]]);

export const evmNetworkChainToEvmChainId = constMap(networkChainEvmCIdEntries);
