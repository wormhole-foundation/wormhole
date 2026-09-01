import { Platform } from "@wormhole-foundation/sdk-base";

/**
 * Chains the SDK 6 chain list dropped entirely — a sunset bridge (Oasis,
 * Aurora, Fantom, Karura, Acala, Neon, Gnosis, Rootstock, Scroll, Mantle,
 * Blast, Xlayer, Snaxchain) or a migration to a new chain id (Terra ->
 * Terra2, Xpla's own SDK removed). Unlike Terra2 (../terra2), the CLI does
 * not keep these alive for execution: this map exists only so that ids and
 * names which still surface in historical VAAs and on-chain registration
 * state (e.g. an old TokenBridge registration for Blast) can be converted
 * instead of crashing the SDK's chain-id assertions.
 */
export const DEPRECATED_CHAINS = {
  Terra: { id: 3, platform: "Cosmwasm" },
  Oasis: { id: 7, platform: "Evm" },
  Aurora: { id: 9, platform: "Evm" },
  Fantom: { id: 10, platform: "Evm" },
  Karura: { id: 11, platform: "Evm" },
  Acala: { id: 12, platform: "Evm" },
  Neon: { id: 17, platform: "Evm" },
  Gnosis: { id: 25, platform: "Evm" },
  Xpla: { id: 28, platform: "Cosmwasm" },
  Rootstock: { id: 33, platform: "Evm" },
  Scroll: { id: 34, platform: "Evm" },
  Mantle: { id: 35, platform: "Evm" },
  Blast: { id: 36, platform: "Evm" },
  Xlayer: { id: 37, platform: "Evm" },
  Snaxchain: { id: 43, platform: "Evm" },
} as const;

export type DeprecatedChain = keyof typeof DEPRECATED_CHAINS;
export type DeprecatedChainId = typeof DEPRECATED_CHAINS[DeprecatedChain]["id"];
export type DeprecatedChainLike = DeprecatedChain | DeprecatedChainId;

const idToDeprecatedChain = new Map<number, DeprecatedChain>(
  (
    Object.entries(DEPRECATED_CHAINS) as [
      DeprecatedChain,
      typeof DEPRECATED_CHAINS[DeprecatedChain]
    ][]
  ).map(([name, { id }]) => [id, name])
);

export const isDeprecatedChain = (chain: unknown): chain is DeprecatedChain =>
  typeof chain === "string" && chain in DEPRECATED_CHAINS;

export const isDeprecatedChainId = (
  chainId: unknown
): chainId is DeprecatedChainId =>
  typeof chainId === "number" && idToDeprecatedChain.has(chainId);

export const isDeprecatedChainLike = (
  chain: unknown
): chain is DeprecatedChainLike =>
  isDeprecatedChain(chain) || isDeprecatedChainId(chain);

export const deprecatedChainIdToName = (
  chainId: DeprecatedChainId
): DeprecatedChain => idToDeprecatedChain.get(chainId)!;

export const deprecatedChainToId = (
  chain: DeprecatedChain
): DeprecatedChainId => DEPRECATED_CHAINS[chain].id;

export const deprecatedChainToPlatform = (chain: DeprecatedChain): Platform =>
  DEPRECATED_CHAINS[chain].platform;
