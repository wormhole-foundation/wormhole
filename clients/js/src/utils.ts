import {
  Chain,
  ChainId,
  Network,
  Platform,
  PlatformToChains,
  assertChainId,
  chainIdToChain,
  chainToChainId,
  chainToPlatform,
  chains,
  contracts,
  toChain,
} from "@wormhole-foundation/sdk-base";
import { spawnSync } from "child_process";
import { ethers } from "ethers";
import {
  TERRA2,
  TERRA2_CHAIN_ID,
  TERRA2_CONNECTIONS,
  Terra2,
  Terra2Like,
  isTerra2Like,
  terra2Contracts,
} from "./chains/terra2/consts";
import {
  DEPRECATED_CHAINS,
  DeprecatedChain,
  DeprecatedChainLike,
  deprecatedChainIdToName,
  deprecatedChainToId,
  deprecatedChainToPlatform,
  isDeprecatedChain,
  isDeprecatedChainLike,
} from "./chains/deprecated/consts";
import { NETWORKS } from "./consts";

/**
 * A chain the CLI supports: everything the SDK knows, plus Terra2 (a full
 * compatibility layer, see ./chains/terra2) and the chains the SDK dropped
 * entirely but whose ids/names can still surface in historical VAAs and
 * on-chain state (see ./chains/deprecated/consts).
 */
export type CliChain = Chain | Terra2 | DeprecatedChain;

/**
 * Anything `toCliChain`/`cliChainToPlatform` can normalize to a `CliChain`:
 * a chain name, a numeric chain id, or either's Terra2/deprecated-chain
 * equivalent.
 */
export type CliChainLike =
  | ChainId
  | CliChain
  | Terra2Like
  | DeprecatedChainLike;

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
  chain: CliChainLike
): asserts chain is PlatformToChains<"Evm"> {
  if (cliChainToPlatform(chain) !== "Evm") {
    throw Error(`Expected an EVM chain, but ${chain} is not`);
  }
}

export function cliChainToChainId(chain: CliChain): number {
  if (chain === TERRA2) {
    return TERRA2_CHAIN_ID;
  }
  if (isDeprecatedChain(chain)) {
    return deprecatedChainToId(chain);
  }
  return chainToChainId(chain);
}

export function cliChainIdToChain(chainId: number): CliChain {
  if (isTerra2Like(chainId)) {
    return TERRA2;
  }
  if (isDeprecatedChainLike(chainId)) {
    return deprecatedChainIdToName(chainId);
  }
  assertChainId(chainId);
  return chainIdToChain(chainId);
}

/**
 * Normalize a chain name or id — including Terra2's and the ids the SDK
 * dropped entirely (see ./chains/deprecated/consts) — to a CliChain name.
 */
export function toCliChain(chain: CliChainLike): CliChain {
  if (isTerra2Like(chain)) {
    return TERRA2;
  }
  if (isDeprecatedChainLike(chain)) {
    return typeof chain === "number" ? deprecatedChainIdToName(chain) : chain;
  }
  return toChain(chain);
}

export function cliChainToPlatform(chain: CliChainLike): Platform {
  if (isTerra2Like(chain)) {
    return "Cosmwasm";
  }
  if (isDeprecatedChainLike(chain)) {
    return deprecatedChainToPlatform(
      typeof chain === "number" ? deprecatedChainIdToName(chain) : chain
    );
  }
  return chainToPlatform(toChain(chain));
}

/**
 * Every chain the CLI supports: the SDK's list, Terra2 (which the SDK
 * removed but the CLI keeps alive, see ./chains/terra2), and the chains the
 * SDK dropped entirely (see ./chains/deprecated/consts).
 */
export const CLI_CHAINS: CliChain[] = [
  ...chains,
  TERRA2,
  ...(Object.keys(DEPRECATED_CHAINS) as DeprecatedChain[]),
];

// Per-chain config lookups that hide the Terra2 split and the deprecated
// chains: the SDK no longer carries their rpc/contracts, so these either
// fall back to the compat layer (Terra2) or report as unconfigured
// (deprecated chains — the CLI never executes against them).

export function getChainRpc(
  network: Network,
  chain: CliChain
): string | undefined {
  if (chain === TERRA2) {
    return TERRA2_CONNECTIONS[network].rpc;
  }
  if (isDeprecatedChain(chain)) {
    return undefined;
  }
  return NETWORKS[network][chain].rpc;
}

export function getCoreContract(
  network: Network,
  chain: CliChain
): string | undefined {
  if (chain === TERRA2) {
    return terra2Contracts(network).core;
  }
  if (isDeprecatedChain(chain)) {
    return undefined;
  }
  return contracts.coreBridge.get(network, chain);
}

export function getTokenBridgeContract(
  network: Network,
  chain: CliChain
): string | undefined {
  if (chain === TERRA2) {
    return terra2Contracts(network).tokenBridge;
  }
  if (isDeprecatedChain(chain)) {
    return undefined;
  }
  return contracts.tokenBridge.get(network, chain);
}

export function getNftBridgeContract(
  network: Network,
  chain: CliChain
): string | undefined {
  // Terra2 never had an NFT bridge deployment; deprecated chains' contracts
  // aren't tracked by the CLI at all.
  if (chain === TERRA2 || isDeprecatedChain(chain)) {
    return undefined;
  }
  return contracts.nftBridge.get(network, chain);
}

export function getRelayerContract(
  network: Network,
  chain: CliChain
): string | undefined {
  if (chain === TERRA2 || isDeprecatedChain(chain)) {
    return undefined;
  }
  return contracts.relayer.get(network, chain);
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

/**
 * Reject chains the SDK dropped entirely: their ids/names still surface in
 * historical VAAs and on-chain state (so the CLI can *name* them), but they
 * have no live bridge to execute against.
 */
export function assertLiveChain(
  chain: CliChain,
  unsupported: string
): asserts chain is Exclude<CliChain, DeprecatedChain> {
  if (isDeprecatedChain(chain)) {
    throw new Error(
      `${chain} was dropped from the SDK and has no live bridge; ${unsupported}`
    );
  }
}

export function chainToCliChain(input: string): CliChain {
  const match = CLI_CHAINS.find(
    (chain) => chain.toLowerCase() === input.toLowerCase()
  );
  if (!match) {
    throw new Error(`Invalid chain: ${input}`);
  }
  return match;
}
