import { CONTRACTS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { Network } from "@wormhole-foundation/sdk-base";

/**
 * Terra2 compatibility layer.
 *
 * @wormhole-foundation/sdk 6.x removed Terra2 (chain id 18) entirely, but the
 * CLI keeps supporting it for governance operations and CosmWasm test
 * coverage. This module carries the chain identity the SDK no longer
 * provides; command handlers special-case Terra2 before converting chain
 * names/ids with the SDK.
 */
export const TERRA2 = "Terra2" as const;
export type Terra2 = typeof TERRA2;

export const TERRA2_CHAIN_ID = 18;

/** "Terra2" or its wormhole chain id. */
export type Terra2Like = Terra2 | typeof TERRA2_CHAIN_ID;

export const isTerra2 = (chain: unknown): chain is Terra2 => chain === TERRA2;

export const isTerra2Like = (chain: unknown): chain is Terra2Like =>
  chain === TERRA2 || chain === TERRA2_CHAIN_ID;

export const TERRA2_NATIVE_DENOM = "uluna";
export const TERRA2_ADDRESS_PREFIX = "terra";

export type Terra2Connection = {
  rpc: string | undefined;
  chain_id: string;
  key: string | undefined;
};

const getEnvVar = (varName: string): string | undefined => process.env[varName];

export const TERRA2_CONNECTIONS: { [network in Network]: Terra2Connection } = {
  Mainnet: {
    rpc: "https://phoenix-lcd.terra.dev",
    chain_id: "phoenix-1",
    key: getEnvVar("TERRA_MNEMONIC"),
  },
  Testnet: {
    rpc: "https://pisco-lcd.terra.dev",
    chain_id: "pisco-1",
    key: getEnvVar("TERRA_MNEMONIC_TESTNET"),
  },
  Devnet: {
    rpc: "http://localhost:1318",
    chain_id: "phoenix-1",
    key: "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius",
  },
};

// The current SDK dropped the Terra2 contract addresses along with the chain;
// the frozen legacy SDK still has the canonical values.
const legacyNetwork = {
  Mainnet: "MAINNET",
  Testnet: "TESTNET",
  Devnet: "DEVNET",
} as const;

export const terra2Contracts = (
  network: Network
): { core: string; tokenBridge: string } => {
  const c = CONTRACTS[legacyNetwork[network]].terra2;
  return { core: c.core, tokenBridge: c.token_bridge };
};
