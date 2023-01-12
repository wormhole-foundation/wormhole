export type Network = "MAINNET" | "TESTNET" | "DEVNET"

export function assertNetwork(n: string): asserts n is Network {
    if (
      n !== "MAINNET" &&
      n !== "TESTNET" &&
      n !== "DEVNET"
    ) {
      throw Error(`Unknown network: ${n}`);
    }
  }
