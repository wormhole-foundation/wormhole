import { ChainName } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { config } from "dotenv";
import { homedir } from "os";

config({ path: `${homedir()}/.wormhole/.env` });

const getEnvVar = (varName: string): string | undefined => process.env[varName];

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
    key: getEnvVar("SOLANA_KEY"),
  },
  terra: {
    rpc: "https://lcd.terra.dev",
    chain_id: "columbus-5",
    key: getEnvVar("TERRA_MNEMONIC"),
  },
  ethereum: {
    rpc: `https://rpc.ankr.com/eth`,
    key: getEnvVar("ETH_KEY"),
    chain_id: 1,
  },
  bsc: {
    rpc: "https://bsc-dataseed.binance.org/",
    key: getEnvVar("ETH_KEY"),
    chain_id: 56,
  },
  polygon: {
    rpc: "https://rpc.ankr.com/polygon",
    key: getEnvVar("ETH_KEY"),
    chain_id: 137,
  },
  avalanche: {
    rpc: "https://rpc.ankr.com/avalanche",
    key: getEnvVar("ETH_KEY"),
    chain_id: 43114,
  },
  algorand: {
    rpc: "https://mainnet-api.algonode.cloud",
    key: getEnvVar("ALGORAND_KEY"),
  },
  oasis: {
    rpc: "https://emerald.oasis.dev/",
    key: getEnvVar("ETH_KEY"),
    chain_id: 42262,
  },
  fantom: {
    rpc: "https://rpc.ftm.tools/",
    key: getEnvVar("ETH_KEY"),
    chain_id: 250,
  },
  aurora: {
    rpc: "https://mainnet.aurora.dev",
    key: getEnvVar("ETH_KEY"),
    chain_id: 1313161554,
  },
  karura: {
    rpc: "https://eth-rpc-karura.aca-api.network/",
    key: getEnvVar("ETH_KEY"),
    chain_id: 686,
  },
  acala: {
    rpc: "https://eth-rpc-acala.aca-api.network/",
    key: getEnvVar("ETH_KEY"),
    chain_id: 787,
  },
  klaytn: {
    rpc: "https://public-node-api.klaytnapi.com/v1/cypress",
    key: getEnvVar("ETH_KEY"),
    chain_id: 8217,
  },
  celo: {
    rpc: "https://forno.celo.org",
    key: getEnvVar("ETH_KEY"),
    chain_id: 42220,
  },
  near: {
    rpc: "https://rpc.mainnet.near.org",
    key: getEnvVar("NEAR_KEY"),
    networkId: "mainnet",
  },
  injective: {
    rpc: "http://sentry0.injective.network:26657",
    chain_id: "injective-1",
    key: getEnvVar("INJECTIVE_KEY"),
  },
  osmosis: {
    rpc: undefined,
    key: undefined,
  },
  aptos: {
    rpc: "https://fullnode.mainnet.aptoslabs.com/v1",
    key: getEnvVar("APTOS_KEY"),
  },
  sui: {
    rpc: "https://fullnode.mainnet.sui.io:443",
    key: getEnvVar("SUI_KEY"),
  },
  pythnet: {
    rpc: "http://api.pythnet.pyth.network:8899/",
    key: getEnvVar("SOLANA_KEY"),
  },
  xpla: {
    rpc: "https://dimension-lcd.xpla.dev",
    chain_id: "dimension_37-1",
    key: getEnvVar("XPLA_KEY"),
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
    key: getEnvVar("ETH_KEY"),
    chain_id: 1284,
  },
  neon: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  terra2: {
    rpc: "https://phoenix-lcd.terra.dev",
    chain_id: "phoenix-1",
    key: getEnvVar("TERRA_MNEMONIC"),
  },
  arbitrum: {
    rpc: "https://arb1.arbitrum.io/rpc",
    key: getEnvVar("ETH_KEY"),
    chain_id: 42161,
  },
  optimism: {
    rpc: "https://mainnet.optimism.io",
    key: getEnvVar("ETH_KEY"),
    chain_id: 10,
  },
  gnosis: {
    rpc: "https://rpc.gnosischain.com/",
    key: getEnvVar("ETH_KEY"),
    chain_id: 100,
  },
  base: {
    rpc: "https://mainnet.base.org",
    key: getEnvVar("ETH_KEY"),
    chain_id: 8453,
  },
  sei: {
    rpc: "https://sei-rpc.polkachu.com/",
    key: getEnvVar("SEI_KEY"),
  },
  rootstock: {
    rpc: "https://public-node.rsk.co",
    key: getEnvVar("ETH_KEY"),
    chain_id: 30,
  },
  scroll: {
    rpc: "https://rpc.ankr.com/scroll",
    key: getEnvVar("ETH_KEY"),
    chain_id: 534352,
  },
  mantle: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  blast: {
    rpc: "https://rpc.ankr.com/blast",
    key: getEnvVar("ETH_KEY"),
    chain_id: 81457,
  },
  xlayer: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  linea: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  berachain: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  seievm: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  sepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  holesky: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  cosmoshub: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  evmos: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  kujira: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  neutron: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  celestia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  arbitrum_sepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  base_sepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  optimism_sepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  polygon_sepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  stargaze: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  seda: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  dymension: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  provenance: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
};

const TESTNET = {
  unset: {
    rpc: undefined,
    key: undefined,
  },
  solana: {
    rpc: "https://api.devnet.solana.com",
    key: getEnvVar("SOLANA_KEY_TESTNET"),
  },
  terra: {
    rpc: "https://bombay-lcd.terra.dev",
    chain_id: "bombay-12",
    key: getEnvVar("TERRA_MNEMONIC_TESTNET"),
  },
  ethereum: {
    rpc: `https://rpc.ankr.com/eth_goerli`,
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 5,
  },
  bsc: {
    rpc: "https://data-seed-prebsc-1-s1.binance.org:8545",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 97,
  },
  polygon: {
    rpc: `https://rpc.ankr.com/polygon_mumbai`,
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 80001,
  },
  avalanche: {
    rpc: "https://rpc.ankr.com/avalanche_fuji",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 43113,
  },
  oasis: {
    rpc: "https://testnet.emerald.oasis.dev",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 42261,
  },
  algorand: {
    rpc: "https://testnet-api.algonode.cloud",
    key: getEnvVar("ALGORAND_KEY_TESTNET"),
  },
  fantom: {
    rpc: "https://rpc.testnet.fantom.network",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 4002,
  },
  aurora: {
    rpc: "https://testnet.aurora.dev",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 1313161555,
  },
  karura: {
    rpc: "https://eth-rpc-karura-testnet.aca-staging.network",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 596,
  },
  acala: {
    rpc: "https://eth-rpc-acala-testnet.aca-staging.network",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 595,
  },
  klaytn: {
    rpc: "https://api.baobab.klaytn.net:8651",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 1001,
  },
  celo: {
    rpc: "https://alfajores-forno.celo-testnet.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 44787,
  },
  near: {
    rpc: "https://rpc.testnet.near.org",
    key: getEnvVar("NEAR_KEY_TESTNET"),
    networkId: "testnet",
  },
  injective: {
    rpc: "https://k8s.testnet.tm.injective.network:443",
    chain_id: "injective-888",
    key: getEnvVar("INJECTIVE_KEY_TESTNET"),
  },
  osmosis: {
    rpc: undefined,
    chain_id: "osmo-test-4",
    key: getEnvVar("OSMOSIS_KEY_TESTNET"),
  },
  aptos: {
    rpc: "https://fullnode.testnet.aptoslabs.com/v1",
    key: getEnvVar("APTOS_TESTNET"),
  },
  sui: {
    rpc: "https://fullnode.testnet.sui.io:443",
    key: getEnvVar("SUI_KEY_TESTNET"),
  },
  pythnet: {
    rpc: "https://api.pythtest.pyth.network/",
    key: getEnvVar("SOLANA_KEY_TESTNET"),
  },
  xpla: {
    rpc: "https://cube-lcd.xpla.dev:443",
    chain_id: "cube_47-5",
    key: getEnvVar("XPLA_KEY_TESTNET"),
  },
  sei: {
    rpc: "https://rpc.atlantic-2.seinetwork.io",
    key: getEnvVar("SEI_KEY_TESTNET"),
  },
  scroll: {
    rpc: "https://rpc.ankr.com/scroll_sepolia_testnet",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 534353,
  },
  mantle: {
    rpc: "https://mantle-sepolia.drpc.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 5003,
  },
  blast: {
    rpc: "https://blast-sepolia.drpc.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 168587773,
  },
  xlayer: {
    rpc: "https://testrpc.xlayer.tech/",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 195,
  },
  linea: {
    rpc: "https://rpc.sepolia.linea.build",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 59141,
  },
  berachain: {
    rpc: "https://artio.rpc.berachain.com/",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 80085,
  },
  seievm: {
    rpc: "https://evm-rpc-arctic-1.sei-apis.com/",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 713715,
  },
  sepolia: {
    rpc: "https://rpc.ankr.com/eth_sepolia",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 11155111,
  },
  holesky: {
    rpc: "https://rpc.ankr.com/eth_holesky",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 17000,
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
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 1287,
  },
  neon: {
    rpc: "https://proxy.devnet.neonlabs.org/solana",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: undefined,
  },
  terra2: {
    rpc: "https://pisco-lcd.terra.dev",
    chain_id: "pisco-1",
    key: getEnvVar("TERRA_MNEMONIC_TESTNET"),
  },
  arbitrum: {
    rpc: "https://goerli-rollup.arbitrum.io/rpc",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 421613,
  },
  optimism: {
    rpc: "https://goerli.optimism.io",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 420,
  },
  gnosis: {
    rpc: "https://sokol.poa.network/",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 77,
  },
  base: {
    rpc: "https://goerli.base.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 84531,
  },
  rootstock: {
    rpc: "https://public-node.testnet.rsk.co",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 31,
  },
  cosmoshub: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  evmos: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  kujira: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  neutron: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  celestia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  arbitrum_sepolia: {
    rpc: "https://arbitrum-sepolia.publicnode.com",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 421614,
  },
  base_sepolia: {
    rpc: "https://sepolia.base.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 84532,
  },
  optimism_sepolia: {
    rpc: "https://rpc.ankr.com/optimism_sepolia",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 11155420,
  },
  polygon_sepolia: {
    rpc: "https://rpc-amoy.polygon.technology/",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 80002,
  },
  stargaze: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  seda: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  dymension: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  provenance: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
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
    key: getEnvVar("ALGORAND_KEY_DEVNET"),
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
  sei: {
    rpc: undefined,
    key: undefined,
  },
  scroll: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  mantle: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  blast: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  xlayer: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  linea: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  berachain: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  seievm: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  sepolia: {
    rpc: undefined,
    key: undefined,
  },
  holesky: {
    rpc: undefined,
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
    rpc: "http://0.0.0.0:9000",
    key: "AGA20wtGcwbcNAG4nwapbQ5wIuXwkYQEWFUoSVAxctHb",
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
  rootstock: {
    rpc: undefined,
    key: undefined,
  },
  cosmoshub: {
    rpc: undefined,
    key: undefined,
  },
  evmos: {
    rpc: undefined,
    key: undefined,
  },
  kujira: {
    rpc: undefined,
    key: undefined,
  },
  neutron: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  celestia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  arbitrum_sepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  base_sepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  optimism_sepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  polygon_sepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  stargaze: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  seda: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  dymension: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  provenance: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
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
