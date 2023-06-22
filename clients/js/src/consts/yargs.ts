import {
  CHAINS,
  ChainId,
  ChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";

export const DEBUG_OPTIONS = {
  alias: "d",
  describe: "Log debug info",
  type: "boolean",
  demandOption: false,
} as const;

export const NAMED_ADDRESSES_OPTIONS = {
  describe: "Named addresses in the format addr1=0x0,addr2=0x1,...",
  type: "string",
  demandOption: false,
} as const;

export const NETWORK_OPTIONS = {
  alias: "n",
  describe: "Network",
  choices: ["mainnet", "testnet", "devnet"],
  demandOption: true,
} as const;

export const PRIVATE_KEY_OPTIONS = {
  alias: "k",
  describe: "Custom private key to sign transactions",
  demandOption: false,
  type: "string",
} as const;

export const RPC_OPTIONS = {
  alias: "r",
  describe: "Override default rpc endpoint url",
  type: "string",
  demandOption: false,
} as const;

export const CHAIN_ID_OR_NAME_CHOICES = [
  ...Object.keys(CHAINS),
  ...Object.values(CHAINS),
] as (ChainName | ChainId)[];

export const CHAIN_NAME_CHOICES = Object.keys(CHAINS).filter(
  (c) => c !== "unset"
) as ChainName[];
