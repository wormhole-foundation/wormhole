import { spawnSync } from "child_process";
import { ethers } from "ethers";

export type Network = "MAINNET" | "TESTNET" | "DEVNET";

export function assertNetwork(n: string): asserts n is Network {
  if (n !== "MAINNET" && n !== "TESTNET" && n !== "DEVNET") {
    throw Error(`Unknown network: ${n}`);
  }
}

export const checkBinary = (binaryName: string, readmeUrl?: string): void => {
  const binary = spawnSync(binaryName, ["--version"]);
  if (binary.status !== 0) {
    console.error(
      `${binaryName} is not installed. Please install ${binaryName} and try again.`
    );
    if (readmeUrl) {
      console.error(`See ${readmeUrl} for instructions.`);
    }
    process.exit(1);
  }
};

export const evm_address = (x: string): string => {
  return hex(x).substring(2).padStart(64, "0");
};

export const hex = (x: string): string => {
  return ethers.utils.hexlify(x, { allowMissingPrefix: true });
};
