import { Network } from "./networks";
import { ChainName } from "./chains";
import { constMap, RoArray } from "../utils";

const shareFinalities = [
  ["Ethereum", 64],
  ["Solana", 32],
  ["Polygon", 64],
  ["Bsc", 15],
  ["Avalanche", 1],
  ["Fantom", 1],
  ["Celo", 1],
  ["Moonbeam", 1],
  ["Sui", 0],
  ["Aptos", 0],
  ["Sei", 0],
] as const;

const finalityThresholds = [
  [
    "Mainnet",
    [
      ["Ethereum", 64],
      ["Solana", 32],
      ["Polygon", 512],
      ["Bsc", 15],
      ["Avalanche", 1],
      ["Fantom", 1],
      ["Celo", 1],
      ["Moonbeam", 1],
      ["Sui", 0],
      ["Aptos", 0],
      ["Sei", 0],
    ],
  ],
  ["Testnet", shareFinalities],
  ["Devnet", shareFinalities],
] as const satisfies RoArray<readonly [Network, RoArray<readonly [ChainName, number]>]>;

export const finalityThreshold = constMap(finalityThresholds);

const blockTimeMilliseconds = [
  ["Acala", 12000],
  ["Algorand", 3300],
  ["Aptos", 4000],
  ["Arbitrum", 300],
  ["Aurora", 3000],
  ["Avalanche", 2000],
  ["Base", 2000],
  ["Bsc", 3000],
  ["Celo", 5000],
  ["Cosmoshub", 5000],
  ["Ethereum", 15000],
  ["Evmos", 2000],
  ["Fantom", 2500],
  ["Gnosis", 5000],
  ["Injective", 2500],
  ["Karura", 12000],
  ["Klaytn", 1000],
  ["Kujira", 3000],
  ["Moonbeam", 12000],
  ["Near", 1500],
  ["Neon", 30000],
  ["Oasis", 6000],
  ["Optimism", 2000],
  ["Osmosis", 6000],
  ["Polygon", 2000],
  ["Rootstock", 30000],
  ["Sei", 400],
  ["Sepolia", 15000],
  ["Solana", 400],
  ["Sui", 3000],
  ["Terra", 6000],
  ["Terra2", 6000],
  ["Xpla", 5000],
  ["Wormchain", 5000],
  ["Btc", 600000],
  ["Pythnet", 400],
] as const satisfies RoArray<readonly [ChainName, number]>;

export const blockTime = constMap(blockTimeMilliseconds);
