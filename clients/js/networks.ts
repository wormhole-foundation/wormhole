import { ChainName } from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

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
    rpc: "https://mainnet-api.algonode.cloud",
    key: get_env_var("ALGORAND_KEY"),
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
    rpc: "https://rpc.mainnet.near.org",
    key: get_env_var("NEAR_KEY"),
    networkId: "mainnet",
    deployerAccount:
      "85957f38de1768d6db9eab29bee9dd2a01462aff9c8d83daefb9bcd2506c32d2",
  },
  injective: {
    rpc: "http://sentry0.injective.network:26657",
    chain_id: "injective-1",
    key: get_env_var("INJECTIVE_KEY"),
  },
  osmosis: {
    rpc: undefined,
    key: undefined,
  },
  aptos: {
    rpc: "https://fullnode.mainnet.aptoslabs.com/v1",
    key: get_env_var("APTOS_KEY"),
  },
  sui: {
    rpc: undefined,
    key: undefined,
  },
  pythnet: {
    rpc: "http://api.pythnet.pyth.network:8899/",
    key: get_env_var("SOLANA_KEY"),
  },
  xpla: {
    rpc: "https://dimension-lcd.xpla.dev",
    chain_id: "dimension_37-1",
    key: get_env_var("XPLA_KEY"),
  },
  btc: {
    rpc: undefined,
    key: undefined,
  },
  wormchain: {
    rpc: undefined,
    key: undefined,
  },
  moonbeam: {
    rpc: "https://rpc.api.moonbeam.network",
    key: get_env_var("ETH_KEY"),
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
  arbitrum: {
    rpc: "https://arb1.arbitrum.io/rpc",
    key: get_env_var("ETH_KEY"),
  },
  optimism: {
    rpc: "https://mainnet.optimism.io",
    key: get_env_var("ETH_KEY"),
  },
  gnosis: {
    rpc: "https://rpc.gnosischain.com/",
    key: get_env_var("ETH_KEY"),
  },
  base: {
    rpc: undefined,
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
    key: get_env_var("SOLANA_KEY_TESTNET"),
  },
  terra: {
    rpc: "https://bombay-lcd.terra.dev",
    chain_id: "bombay-12",
    key: get_env_var("TERRA_MNEMONIC_TESTNET"),
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
    rpc: "https://testnet-api.algonode.cloud",
    key: get_env_var("ALGORAND_KEY_TESTNET"),
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
    rpc: "https://eth-rpc-karura-testnet.aca-staging.network",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  acala: {
    rpc: "https://eth-rpc-acala-testnet.aca-staging.network",
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
    rpc: "https://rpc.testnet.near.org",
    key: get_env_var("NEAR_KEY_TESTNET"),
    networkId: "testnet",
    deployerAccount: "wormhole.testnet",
  },
  injective: {
    rpc: "https://k8s.testnet.tm.injective.network:443",
    chain_id: "injective-888",
    key: get_env_var("INJECTIVE_KEY_TESTNET"),
  },
  osmosis: {
    rpc: undefined,
    chain_id: "osmo-test-4",
    key: get_env_var("OSMOSIS_KEY_TESTNET"),
  },
  aptos: {
    rpc: "https://fullnode.testnet.aptoslabs.com/v1",
    key: get_env_var("APTOS_TESTNET"),
  },
  sui: {
    rpc: undefined,
    key: undefined,
  },
  pythnet: {
    rpc: "https://api.pythtest.pyth.network/",
    key: get_env_var("SOLANA_KEY_TESTNET"),
  },
  xpla: {
    rpc: "https://cube-lcd.xpla.dev:443",
    chain_id: "cube_47-5",
    key: get_env_var("XPLA_KEY_TESTNET"),
  },
  btc: {
    rpc: undefined,
    key: undefined,
  },
  wormchain: {
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
    key: get_env_var("TERRA_MNEMONIC_TESTNET"),
  },
  arbitrum: {
    rpc: "https://goerli-rollup.arbitrum.io/rpc",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  optimism: {
    rpc: "https://goerli.optimism.io",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  gnosis: {
    rpc: "https://sokol.poa.network/",
    key: get_env_var("ETH_KEY_TESTNET"),
  },
  base: {
    rpc: "https://goerli.base.org",
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
    rpc: "http://localhost",
    key: get_env_var("ALGORAND_KEY_DEVNET"),
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
    networkId: "sandbox",
    deployerAccount: "test.near",
  },
  injective: {
    rpc: undefined,
    key: undefined,
  },
  osmosis: {
    rpc: undefined,
    key: undefined,
  },
  pythnet: {
    rpc: undefined,
    key: undefined,
  },
  btc: {
    rpc: undefined,
    key: undefined,
  },
  xpla: {
    rpc: undefined,
    chain_id: undefined,
    key: undefined,
  },
  wormchain: {
    rpc: "http://localhost:1319",
    chain_id: "wormchain",
    key: undefined,
  },
  aptos: {
    rpc: "http://0.0.0.0:8080",
    key: "537c1f91e56891445b491068f519b705f8c0f1a1e66111816dd5d4aa85b8113d",
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
  arbitrum: {
    rpc: undefined,
    key: undefined,
  },
  optimism: {
    rpc: undefined,
    key: undefined,
  },
  gnosis: {
    rpc: undefined,
    key: undefined,
  },
  base: {
    rpc: undefined,
    key: undefined,
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
