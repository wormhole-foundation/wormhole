import {
  Chain,
  ChainId,
  Network,
  PlatformToChains,
  chainToPlatform,
  toChain,
} from "@wormhole-foundation/sdk-base";
import { ChainId as OldChainId, ChainName as OldChainName } from "@certusone/wormhole-sdk";
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
  const chainStr = input[0].toUpperCase() + input.slice(1).toLowerCase();
  return toChain(chainStr);
}

/**
 * HACK: Cast ChainId from new SDK to old SDK type.
 *
 * The @certusone/wormhole-sdk package depends on an old version of
 * @wormhole-foundation/sdk-base, which doesn't know about new chain IDs.
 * This causes type errors when passing ChainId values from the new SDK
 * to functions in the old SDK.
 *
 * This function performs an unsafe cast to work around the type incompatibility.
 * It should be removed once @certusone/wormhole-sdk is updated or deprecated.
 */
export function castChainIdToOldSdk(chainId: ChainId): OldChainId {
  return chainId as any as OldChainId;
}

/**
 * HACK: Cast Chain from new SDK to old SDK type.
 *
 * The @certusone/wormhole-sdk package depends on an old version of
 * @wormhole-foundation/sdk-base, which doesn't know about new chains.
 * This causes type errors when passing Chain values from the new SDK
 * to functions in the old SDK.
 *
 * This function performs an unsafe cast to work around the type incompatibility.
 * It should be removed once @certusone/wormhole-sdk is updated or deprecated.
 */
export function castChainToOldSdk(chain: Chain): OldChainName {
  return chain as any as OldChainName;
}
