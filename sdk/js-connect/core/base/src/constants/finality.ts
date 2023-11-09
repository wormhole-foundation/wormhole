import { ChainName } from "./chains";
import { constMap, RoArray } from "../utils";

// https://www.notion.so/Finality-in-Wormhole-78ffa423abd44b7cbe38483a16040d83

// Recognized Consistency Levels for determining when a guardian
// may sign a VAA for a given wormhole message
export enum ConsistencyLevels {
  // SolanaConfirmed = 0,
  Finalized = 1,
  Immediate = 200,
  Safe = 201,
}

// Number of blocks before a transaction is considered "safe"
// In this case its the number of rounds for each epoch, once an epoch
// is completed, the transaction is considered safe
const safeThresholds = [
  ["Ethereum", 32], // number of rounds in an epoch
] as const satisfies RoArray<readonly [ChainName, number]>;
export const safeThreshold = constMap(safeThresholds);

// Number of blocks before a transaction is considered "final"
const finalityThresholds = [
  ["Ethereum", 64],
  ["Solana", 32],
  ["Polygon", 512],
  ["Bsc", 15],
  ["Fantom", 1],
  ["Celo", 1],
  ["Moonbeam", 1],
  ["Avalanche", 0],
  ["Sui", 0],
  ["Algorand", 0],
  ["Aptos", 0],
  ["Sei", 0],
  ["Near", 0],
  ["Terra", 0],
  ["Terra2", 0],
  ["Xpla", 0],
  ["Injective", 0],
] as const satisfies RoArray<readonly [ChainName, number]>;
export const finalityThreshold = constMap(finalityThresholds);

// number of milliseconds between blocks
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

// Estimate the block number that a VAA might be available
// for a given chain, initial block where the tx was submitted
// and consistency level
export function consistencyLevelToBlock(
  chain: ChainName,
  consistencyLevel: number,
  fromBlock: bigint = 0n,
): bigint {
  // We're done
  if (consistencyLevel === ConsistencyLevels.Immediate) return fromBlock;

  // Bsc is the only chain that treats consistency level as # of blocks
  if (chain === "Bsc") return fromBlock + BigInt(consistencyLevel);

  // On Solana 0 is "confirmed", for now just return fromBlock since we
  // have no way of estimating when 66% of the network will have confirmed
  if (chain === "Solana" && consistencyLevel === 0) return fromBlock;

  // Get the number of blocks until finalized
  const chainFinality = finalityThreshold.get(chain);
  if (chainFinality === undefined) throw new Error("Cannot find chain finality for " + chain);

  // If chain finality threshold is 0, it's always instant
  if (chainFinality === 0) return fromBlock;

  // We've handled all the other cases so anything != safe is `finalized`
  if (consistencyLevel !== ConsistencyLevels.Safe) return fromBlock + BigInt(chainFinality);

  // We're only in Safe mode now
  const safeRounds = safeThreshold.get(chain);
  if (safeRounds === undefined) throw new Error("Cannot find safe threshold for " + chain);

  switch (chain) {
    case "Ethereum":
      // On Ethereum "safe" is 1 epoch
      // return the number of blocks until the end
      // of the current epoch
      // 0 is end of epoch, 1 is start
      const epochPosition = fromBlock % BigInt(safeRounds);
      const blocksUntilEndOfEpoch = epochPosition === 0n ? 0n : BigInt(safeRounds) - epochPosition;
      return fromBlock + blocksUntilEndOfEpoch;

    default:
      throw new Error("Only Ethereum safe is supported for now");
  }
}
