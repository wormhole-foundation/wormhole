import { Chain } from "@wormhole-foundation/sdk-base";
import { config } from "dotenv";
import { homedir } from "os";

config({ path: `${homedir()}/.wormhole/.env` });

const getEnvVar = (varName: string): string | undefined => process.env[varName];

export type Connection = {
  rpc: string | undefined;
  key: string | undefined;
};

export type ChainConnections = {
  [chain in Chain]: Connection;
};

const Mainnet = {
  Solana: {
    rpc: "https://api.mainnet-beta.solana.com",
    key: getEnvVar("SOLANA_KEY"),
  },
  Ethereum: {
    rpc: `https://ethereum-rpc.publicnode.com`,
    key: getEnvVar("ETH_KEY"),
    chain_id: 1,
  },
  Bsc: {
    rpc: "https://bsc-rpc.publicnode.com",
    key: getEnvVar("ETH_KEY"),
    chain_id: 56,
  },
  Polygon: {
    rpc: "https://rpc.ankr.com/polygon",
    key: getEnvVar("ETH_KEY"),
    chain_id: 137,
  },
  Avalanche: {
    rpc: "https://rpc.ankr.com/avalanche",
    key: getEnvVar("ETH_KEY"),
    chain_id: 43114,
  },
  Algorand: {
    rpc: "https://mainnet-api.algonode.cloud",
    key: getEnvVar("ALGORAND_KEY"),
  },
  Fantom: {
    rpc: "https://rpc.ftm.tools/",
    key: getEnvVar("ETH_KEY"),
    chain_id: 250,
  },
  Klaytn: {
    rpc: "https://public-node-api.klaytnapi.com/v1/cypress",
    key: getEnvVar("ETH_KEY"),
    chain_id: 8217,
  },
  Celo: {
    rpc: "https://forno.celo.org",
    key: getEnvVar("ETH_KEY"),
    chain_id: 42220,
  },
  Near: {
    rpc: "https://rpc.mainnet.near.org",
    key: getEnvVar("NEAR_KEY"),
    networkId: "mainnet",
  },
  Injective: {
    rpc: "http://sentry0.injective.network:26657",
    chain_id: "injective-1",
    key: getEnvVar("INJECTIVE_KEY"),
  },
  Osmosis: {
    rpc: undefined,
    key: undefined,
  },
  Aptos: {
    rpc: "https://fullnode.mainnet.aptoslabs.com/v1",
    key: getEnvVar("APTOS_KEY"),
  },
  Sui: {
    rpc: "https://fullnode.mainnet.sui.io:443",
    key: getEnvVar("SUI_KEY"),
  },
  Pythnet: {
    rpc: "http://api.pythnet.pyth.network:8899/",
    key: getEnvVar("SOLANA_KEY"),
  },
  Btc: {
    rpc: undefined,
    key: undefined,
  },
  Wormchain: {
    rpc: undefined,
    key: undefined,
  },
  Moonbeam: {
    rpc: "https://rpc.api.moonbeam.network",
    key: getEnvVar("ETH_KEY"),
    chain_id: 1284,
  },
  Arbitrum: {
    rpc: "https://arb1.arbitrum.io/rpc",
    key: getEnvVar("ETH_KEY"),
    chain_id: 42161,
  },
  Optimism: {
    rpc: "https://mainnet.optimism.io",
    key: getEnvVar("ETH_KEY"),
    chain_id: 10,
  },
  Base: {
    rpc: "https://mainnet.base.org",
    key: getEnvVar("ETH_KEY"),
    chain_id: 8453,
  },
  Sei: {
    rpc: "https://sei-rpc.polkachu.com/",
    key: getEnvVar("SEI_KEY"),
  },
  Scroll: {
    rpc: "https://rpc.ankr.com/scroll",
    key: getEnvVar("ETH_KEY"),
    chain_id: 534352,
  },
  Mantle: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Xlayer: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Linea: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Berachain: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Seievm: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Sepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Holesky: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Cosmoshub: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Evmos: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Kujira: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Neutron: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Celestia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  ArbitrumSepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  BaseSepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  OptimismSepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  PolygonSepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Stargaze: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Seda: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Dymension: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Provenance: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Fogo: {
    rpc: "https://mainnet.fogo.io",
    key: getEnvVar("FOGO_KEY"),
  },
  Unichain: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  HyperCore: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Worldchain: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Ink: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  HyperEVM: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Monad: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Mezo: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Sonic: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Converge: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Plume: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  XRPLEVM: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Plasma: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  CreditCoin: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Noble: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
};

const Testnet = {
  Solana: {
    rpc: "https://api.devnet.solana.com",
    key: getEnvVar("SOLANA_KEY_TESTNET"),
  },
  Ethereum: {
    rpc: `https://rpc.ankr.com/eth_goerli`,
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 5,
  },
  Bsc: {
    rpc: "https://data-seed-prebsc-1-s1.binance.org:8545",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 97,
  },
  Polygon: {
    rpc: `https://rpc.ankr.com/polygon_mumbai`,
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 80001,
  },
  Avalanche: {
    rpc: "https://rpc.ankr.com/avalanche_fuji",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 43113,
  },
  Algorand: {
    rpc: "https://testnet-api.algonode.cloud",
    key: getEnvVar("ALGORAND_KEY_TESTNET"),
  },
  Fantom: {
    rpc: "https://rpc.testnet.fantom.network",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 4002,
  },
  Klaytn: {
    rpc: "https://api.baobab.klaytn.net:8651",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 1001,
  },
  Celo: {
    rpc: "https://alfajores-forno.celo-testnet.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 44787,
  },
  Near: {
    rpc: "https://rpc.testnet.near.org",
    key: getEnvVar("NEAR_KEY_TESTNET"),
    networkId: "testnet",
  },
  Injective: {
    rpc: "https://k8s.testnet.tm.injective.network:443",
    chain_id: "injective-888",
    key: getEnvVar("INJECTIVE_KEY_TESTNET"),
  },
  Osmosis: {
    rpc: undefined,
    chain_id: "osmo-test-4",
    key: getEnvVar("OSMOSIS_KEY_TESTNET"),
  },
  Aptos: {
    rpc: "https://fullnode.testnet.aptoslabs.com/v1",
    key: getEnvVar("APTOS_TESTNET"),
  },
  Sui: {
    rpc: "https://fullnode.testnet.sui.io:443",
    key: getEnvVar("SUI_KEY_TESTNET"),
  },
  Pythnet: {
    rpc: "https://api.pythtest.pyth.network/",
    key: getEnvVar("SOLANA_KEY_TESTNET"),
  },
  Sei: {
    rpc: "https://rpc.atlantic-2.seinetwork.io",
    key: getEnvVar("SEI_KEY_TESTNET"),
  },
  Scroll: {
    rpc: "https://rpc.ankr.com/scroll_sepolia_testnet",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 534353,
  },
  Mantle: {
    rpc: "https://mantle-sepolia.drpc.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 5003,
  },
  Blast: {
    rpc: "https://blast-sepolia.drpc.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 168587773,
  },
  Xlayer: {
    rpc: "https://testrpc.xlayer.tech/",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 195,
  },
  Linea: {
    rpc: "https://rpc.sepolia.linea.build",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 59141,
  },
  Berachain: {
    rpc: "https://bartio.rpc.berachain.com/",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 80084,
  },
  Snaxchain: {
    rpc: "https://testnet.snaxchain.io",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 13001,
  },
  Seievm: {
    rpc: "https://evm-rpc-arctic-1.sei-apis.com/",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 713715,
  },
  Sepolia: {
    rpc: "https://rpc.ankr.com/eth_sepolia",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 11155111,
  },
  Holesky: {
    rpc: "https://rpc.ankr.com/eth_holesky",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 17000,
  },
  Btc: {
    rpc: undefined,
    key: undefined,
  },
  Wormchain: {
    rpc: undefined,
    key: undefined,
  },
  Moonbeam: {
    rpc: "https://rpc.api.moonbase.moonbeam.network",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 1287,
  },
  Arbitrum: {
    rpc: "https://goerli-rollup.arbitrum.io/rpc",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 421613,
  },
  Optimism: {
    rpc: "https://goerli.optimism.io",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 420,
  },
  Base: {
    rpc: "https://goerli.base.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 84531,
  },
  Cosmoshub: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Evmos: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Kujira: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Neutron: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Celestia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  ArbitrumSepolia: {
    rpc: "https://arbitrum-sepolia.publicnode.com",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 421614,
  },
  BaseSepolia: {
    rpc: "https://sepolia.base.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 84532,
  },
  OptimismSepolia: {
    rpc: "https://rpc.ankr.com/optimism_sepolia",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 11155420,
  },
  PolygonSepolia: {
    rpc: "https://rpc-amoy.polygon.technology/",
    key: getEnvVar("ETH_KEY_TESTNET"),
    chain_id: 80002,
  },
  Stargaze: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Seda: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Dymension: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Provenance: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Fogo: {
    rpc: "https://testnet-alt.fogo.io",
    key: getEnvVar("FOGO_KEY_TESTNET"),
  },
  Unichain: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  HyperCore: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Worldchain: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Ink: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  HyperEVM: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Monad: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Mezo: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Sonic: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Converge: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Plume: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  XRPLEVM: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Plasma: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  CreditCoin: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Noble: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
};

const Devnet = {
  Solana: {
    rpc: "http://127.0.0.1:8899",
    key: "J2D4pwDred8P9ioyPEZVLPht885AeYpifsFGUyuzVmiKQosAvmZP4EegaKFrSprBC5vVP1xTvu61vYDWsxBNsYx",
  },
  Ethereum: {
    rpc: "http://localhost:8545",
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  Bsc: {
    rpc: "http://localhost:8546",
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  Polygon: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  Avalanche: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  Algorand: {
    rpc: "http://localhost",
    key: getEnvVar("ALGORAND_KEY_DEVNET"),
  },
  Fantom: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  Klaytn: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  Celo: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  Near: {
    rpc: undefined,
    key: undefined,
    networkId: "sandbox",
  },
  Injective: {
    rpc: undefined,
    key: undefined,
  },
  Osmosis: {
    rpc: undefined,
    key: undefined,
  },
  Pythnet: {
    rpc: undefined,
    key: undefined,
  },
  Btc: {
    rpc: undefined,
    key: undefined,
  },
  Sei: {
    rpc: undefined,
    key: undefined,
  },
  Scroll: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Mantle: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Xlayer: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Linea: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Berachain: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Seievm: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Sepolia: {
    rpc: undefined,
    key: undefined,
  },
  Holesky: {
    rpc: undefined,
    key: undefined,
  },
  Wormchain: {
    rpc: "http://localhost:1319",
    chain_id: "wormchain",
    key: undefined,
  },
  Aptos: {
    rpc: "http://0.0.0.0:8080",
    key: "537c1f91e56891445b491068f519b705f8c0f1a1e66111816dd5d4aa85b8113d",
  },
  Sui: {
    rpc: "http://0.0.0.0:9000",
    key: "AGA20wtGcwbcNAG4nwapbQ5wIuXwkYQEWFUoSVAxctHb",
  },
  Moonbeam: {
    rpc: undefined,
    key: "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d",
  },
  Arbitrum: {
    rpc: undefined,
    key: undefined,
  },
  Optimism: {
    rpc: undefined,
    key: undefined,
  },
  Base: {
    rpc: undefined,
    key: undefined,
  },
  Cosmoshub: {
    rpc: undefined,
    key: undefined,
  },
  Evmos: {
    rpc: undefined,
    key: undefined,
  },
  Kujira: {
    rpc: undefined,
    key: undefined,
  },
  Neutron: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Celestia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  ArbitrumSepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  BaseSepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  OptimismSepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  PolygonSepolia: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Stargaze: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Seda: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Dymension: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Provenance: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Fogo: {
    rpc: undefined,
    key: undefined,
  },
  Unichain: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  HyperCore: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Worldchain: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Ink: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  HyperEVM: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Monad: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Mezo: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Sonic: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Converge: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Plume: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  XRPLEVM: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Plasma: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  CreditCoin: {
    rpc: undefined,
    key: undefined,
    chain_id: undefined,
  },
  Noble: {
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
const isTestnetConnections: ChainConnections = Testnet;

/**
 *
 * See [[isTestnetContracts]]
 */
const isMainnetConnections: ChainConnections = Mainnet;

/**
 *
 * See [[isTestnetContracts]]
 */
const isDevnetConnections: ChainConnections = Devnet;

export const NETWORKS = { Mainnet, Testnet, Devnet };
