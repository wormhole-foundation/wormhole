import {
  CHAIN_ID_SOLANA,
  CONTRACTS as SDK_CONTRACTS,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";

const OVERRIDES = {
  MAINNET: {
    sui: {
      core: "0xaeab97f96cf9877fee2883315d459552b2b921edc16d7ceac6eab944dd88919c",
      token_bridge:
        "0xc57508ee0d4595e5a8728974a4a93a787d38f339757230d441e895422c07aba9",
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
      core: "0x31358d198147da50db32eda2562951d53973a0c0ad5ed738e9b17d88b213d790",
      token_bridge:
        "0x6fb10cdb7aa299e9a4308752dadecb049ff55a892de92992a1edbd7912b3d6da",
    },
    aptos: {
      token_bridge:
        "0x576410486a2da45eee6c949c995670112ddf2fbeedab20350d506328eefc9d4f",
      core: "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
      nft_bridge: undefined,
    },
  },
  DEVNET: {
    sui: {
      core: "0x12253210c90f89e7a8525e6c52d41309ff5bfb31f43f561b5fe6f50cd72f9668", // wormhole module State object ID
      token_bridge:
        "0x830ed228c6f1bcb40003bb49af3277df2cbf933d63a6bcdcb0ba4580a1a7654e", // token_bridge module State object ID
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

export const GOVERNANCE_CHAIN = CHAIN_ID_SOLANA;
export const GOVERNANCE_EMITTER =
  "0000000000000000000000000000000000000000000000000000000000000004";
export const INITIAL_GUARDIAN_DEVNET =
  "befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe";
