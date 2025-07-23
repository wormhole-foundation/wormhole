import {
  Chain,
  ChainId,
  Network,
  PlatformToChains,
  chainToPlatform,
  toChain,
} from "@wormhole-foundation/sdk-base";
import { spawnSync } from "child_process";
import { ethers } from "ethers";

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

export function assertEVMChain(
  chain: ChainId | Chain
): asserts chain is PlatformToChains<"Evm"> {
  if (chainToPlatform(toChain(chain)) !== "Evm") {
    throw Error(`Expected an EVM chain, but ${chain} is not`);
  }
}

export function getNetwork(network: string): Network {
  const lcNetwork: string = network.toLowerCase();
  if (lcNetwork === "mainnet") {
    return "Mainnet";
  }
  if (lcNetwork === "testnet") {
    return "Testnet";
  }
  if (lcNetwork === "devnet") {
    return "Devnet";
  }
  throw new Error(`Unknown network: ${network}`);
}

export function chainToChain(input: string): Chain {
  if (input.length < 2) {
    throw new Error(`Invalid chain: ${input}`);
  }
  let chainStr = input[0].toUpperCase() + input.slice(1).toLowerCase();

  // TODO/Hack: sdk-v1 used "_sepolia" but sdk-v2 uses camel casing. Convert if necessary.
  chainStr = chainStr.replace("_sepolia", "Sepolia");
  return toChain(chainStr);
}
