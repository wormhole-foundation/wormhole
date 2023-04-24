import {
  CHAIN_ID_SOLANA,
  CONTRACTS as SDK_CONTRACTS,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

const OVERRIDES = {
  MAINNET: {
    sui: {
      core: undefined,
      token_bridge: undefined,
    },
    aptos: {
      token_bridge:
        "0x576410486a2da45eee6c949c995670112ddf2fbeedab20350d506328eefc9d4f",
      core: "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
      nft_bridge:
        "0x1bdffae984043833ed7fe223f7af7a3f8902d04129b14f801823e64827da7130",
    },
  },
  TESTNET: {
    sui: {
      core: undefined,
      token_bridge: undefined,
    },
    aptos: {
      token_bridge:
        "0x576410486a2da45eee6c949c995670112ddf2fbeedab20350d506328eefc9d4f",
      core: "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
      nft_bridge:
        "0x1bdffae984043833ed7fe223f7af7a3f8902d04129b14f801823e64827da7130",
    },
  },
  DEVNET: {
    sui: {
      core: "0xfc329a56b1c34688ca285aca200d3f14061ccfd23849e8178c345f50218a2448",
      token_bridge:
        "0x8c8dbbd61276b1fd9fa6b7f7cc966e556b86254d925aa39d28772250a9dc8db2",
    },
    aptos: {
      token_bridge:
        "0x84a5f374d29fc77e370014dce4fd6a55b58ad608de8074b0be5571701724da31",
      core: "0xde0036a9600559e295d5f6802ef6f3f802f510366e0c23912b0655d972166017",
      nft_bridge:
        "0x46da3d4c569388af61f951bdd1153f4c875f90c2991f6b2d0a38e2161a40852c",
    },
  },
};

// TODO(aki): move this to SDK at some point
export const CONTRACTS = {
  MAINNET: { ...SDK_CONTRACTS.MAINNET, ...OVERRIDES.MAINNET },
  TESTNET: { ...SDK_CONTRACTS.TESTNET, ...OVERRIDES.TESTNET },
  DEVNET: { ...SDK_CONTRACTS.DEVNET, ...OVERRIDES.DEVNET },
};

export const NAMED_ADDRESSES_OPTIONS = {
  describe: "Named addresses in the format addr1=0x0,addr2=0x1,...",
  type: "string",
  require: false,
} as const;

export const NETWORK_OPTIONS = {
  alias: "n",
  describe: "Network",
  type: "string",
  choices: ["mainnet", "testnet", "devnet"],
  required: true,
} as const;

export const RPC_OPTIONS = {
  alias: "r",
  describe: "Override default rpc endpoint url",
  type: "string",
  required: false,
} as const;

export const GOVERNANCE_CHAIN = CHAIN_ID_SOLANA;
export const GOVERNANCE_EMITTER =
  "0000000000000000000000000000000000000000000000000000000000000004";
