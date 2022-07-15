import { ChainName } from "@certusone/wormhole-sdk";

require("dotenv").config({ path: `${process.env.HOME}/.wormhole/.env` });

function get_env_var(env: string): string | undefined {
  const v = process.env[env];
  return v;
}

export type Connection = {
  rpc: string | undefined;
  key: string | undefined;
};

export type ChainConnections = {
  [chain in ChainName]: Connection;
};

const MAINNET = {
  unset: {
    rpc: undefined,
    key: undefined,
  },
  solana: {
    rpc: "https://api.mainnet-beta.solana.com",
    key: get_env_var("SOLANA_KEY"),
  },
  terra: {
    rpc: "https://lcd.terra.dev",
    chain_id: "columbus-5",
    key: get_env_var("TERRA_MNEMONIC"),
  },
  ethereum: {
    rpc: `https://rpc.ankr.com/eth`,
    key: get_env_var("ETH_KEY"),
  },
  bsc: {
    rpc: "https://bsc-dataseed.binance.org/",
    key: get_env_var("ETH_KEY"),
  },
  polygon: {
    rpc: "https://rpc.ankr.com/polygon",
    key: get_env_var("ETH_KEY"),
  },
  avalanche: {
    rpc: "https://rpc.ankr.com/avalanche",
    key: get_env_var("ETH_KEY"),
  },
  algorand: {
    rpc: undefined,
    key: undefined,
  },
  oasis: {
    rpc: "https://emerald.oasis.dev/",
    key: get_env_var("ETH_KEY"),
  },
  fantom: {
    rpc: "https://rpc.ftm.tools/",
    key: get_env_var("ETH_KEY"),
  },
  aurora: {
    rpc: "https://mainnet.aurora.dev",
    key: get_env_var("ETH_KEY"),
  },
  karura: {
    rpc: "https://eth-rpc-karura.aca-api.network/",
    key: get_env_var("ETH_KEY"),
  },
  acala: {
    rpc: "https://eth-rpc-acala.aca-api.network/",
    key: get_env_var("ETH_KEY"),
  },
  klaytn: {
    rpc: "https://public-node-api.klaytnapi.com/v1/cypress",
    key: get_env_var("ETH_KEY"),
  },
  celo: {
    rpc: "https://forno.celo.org",
    key: get_env_var("ETH_KEY"),
  },
  near: {
    rpc: undefined,
    key: undefined,
  },
  injective: {
    rpc: undefined,
    key: undefined,
  },
  osmosis: {
    rpc: undefined,
    key: undefined,
  },
  aptos: {
    rpc: undefined,
    key: undefined,
  },
  sui: {
    rpc: undefined,
    key: undefined,
  },
  moonbeam: {
    rpc: undefined,
    key: undefined,
  },
  neon: {
    rpc: undefined,
    key: undefined,
  },
  terra2: {
    rpc: "https://phoenix-lcd.terra.dev",
    chain_id: "phoenix-1",
    key: get_env_var("TERRA_MNEMONIC"),
  },
  ropsten: {
    rpc: `https://rpc.ankr.com/eth_ropsten`,
    key: get_env_var("ETH_KEY"),
  },
};

const TESTNET = {
  unset: {
    rpc: undefined,
    key: undefined,
  },
  solana: {
    rpc: "https://api.devnet.solana.com",
    key: get_env_var("SOLANA_KEY"),
  },
  terra: {
    rpc: "https://bombay-lcd.terra.dev",
    chain_id: "bombay-12",
    key: get_env_var("TERRA_MNEMONIC"),
  },
  ethereum: {
    rpc: `https://rpc.ankr.com/eth_goerli`,
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  bsc: {
    rpc: "https://data-seed-prebsc-1-s1.binance.org:8545",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  polygon: {
    rpc: `https://rpc.ankr.com/polygon_mumbai`,
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  avalanche: {
    rpc: "https://rpc.ankr.com/avalanche_fuji",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  oasis: {
    rpc: "https://testnet.emerald.oasis.dev",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  algorand: {
    rpc: undefined,
    key: undefined,
  },
  fantom: {
    rpc: "https://rpc.testnet.fantom.network",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  aurora: {
    rpc: "https://testnet.aurora.dev",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  karura: {
    rpc: "https://karura-dev.aca-dev.network/eth/http",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  acala: {
    rpc: "https://acala-dev.aca-dev.network/eth/http",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  klaytn: {
    rpc: "https://api.baobab.klaytn.net:8651",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  celo: {
    rpc: "https://alfajores-forno.celo-testnet.org",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  near: {
    rpc: undefined,
    key: undefined,
  },
  injective: {
    rpc: undefined,
    key: undefined,
  },
  osmosis: {
    rpc: undefined,
    key: undefined,
  },
  aptos: {
    rpc: undefined,
    key: undefined,
  },
  sui: {
    rpc: undefined,
    key: undefined,
  },
  moonbeam: {
    rpc: "https://rpc.api.moonbase.moonbeam.network",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  neon: {
    rpc: "https://proxy.devnet.neonlabs.org/solana",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  terra2: {
    rpc: "https://pisco-lcd.terra.dev",
    chain_id: "pisco-1",
    key: get_env_var("TERRA_MNEMONIC"),
  },
  ropsten: {
    rpc: `https://rpc.ankr.com/eth_ropsten`,
    key: get_env_var("ETH_KEY_TESTNET"),
  },
};

const DEVNET = {
  unset: {
    rpc: undefined,
    key: undefined,
  },
  solana: {
    rpc: "http://localhost:8899",
    key: "J2D4pwDred8P9ioyPEZVLPht885AeYpifsFGUyuzVmiKQosAvmZP4EegaKFrSprBC5vVP1xTvu61vYDWsxBNsYx",
  },
  terra: {
    rpc: "http://localhost:1317",
    chain_id: "columbus-5",
    key: "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius",
  },
  ethereum: {
    rpc: "http://localhost:8545",
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  bsc: {
    rpc: "http://localhost:8546",
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  polygon: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  avalanche: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  oasis: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  algorand: {
    rpc: undefined,
    key: undefined,
  },
  fantom: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  aurora: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  karura: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  acala: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  klaytn: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  celo: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  near: {
    rpc: undefined,
    key: undefined,
  },
  injective: {
    rpc: undefined,
    key: undefined,
  },
  osmosis: {
    rpc: undefined,
    key: undefined,
  },
  aptos: {
    rpc: undefined,
    key: undefined,
  },
  sui: {
    rpc: undefined,
    key: undefined,
  },
  moonbeam: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  neon: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  terra2: {
    rpc: "http://localhost:1318",
    chain_id: "phoenix-1",
    key: "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius",
  },
  ropsten: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
};

/**
 *
 * If you get a type error here, it means that a chain you just added does not
 * have an entry in TESTNET.
 * This is implemented as an ad-hoc type assertion instead of a type annotation
 * on TESTNET so that e.g.
 *
 * ```typescript
 * TESTNET['solana'].rpc
 * ```
 * has type 'string' instead of 'string | undefined'.
 *
 * (Do not delete this declaration!)
 */
const isTestnetConnections: ChainConnections = TESTNET;

/**
 *
 * See [[isTestnetContracts]]
 */
const isMainnetConnections: ChainConnections = MAINNET;

/**
 *
 * See [[isTestnetContracts]]
 */
const isDevnetConnections: ChainConnections = DEVNET;

export const NETWORKS = { MAINNET, TESTNET, DEVNET };
