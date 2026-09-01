import {
  Chain,
  Network,
  chainToPlatform,
  chains,
  rpc as sdkRpc,
} from "@wormhole-foundation/sdk-base";
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

/**
 * Every chain known to the SDK gets a default connection: the SDK's default
 * RPC (if any), and the shared EVM key for EVM chains. Chains that need a
 * custom RPC, a dedicated key, or extra fields (e.g. Near's networkId) are
 * overridden explicitly in the per-network objects below.
 */
const defaultConnections = (network: Network): ChainConnections =>
  Object.fromEntries(
    chains.map((chain) => [
      chain,
      {
        rpc: sdkRpc.rpcAddress(network, chain) || undefined,
        key:
          chainToPlatform(chain) === "Evm" && network !== "Devnet"
            ? getEnvVar(network === "Mainnet" ? "ETH_KEY" : "ETH_KEY_TESTNET")
            : undefined,
      },
    ])
  ) as ChainConnections;

const Mainnet = {
  ...defaultConnections("Mainnet"),
  Solana: {
    rpc: "https://api.mainnet-beta.solana.com",
    key: getEnvVar("SOLANA_KEY"),
  },
  Ethereum: {
    rpc: `https://ethereum-rpc.publicnode.com`,
    key: getEnvVar("ETH_KEY"),
  },
  Bsc: {
    rpc: "https://bsc-rpc.publicnode.com",
    key: getEnvVar("ETH_KEY"),
  },
  Polygon: {
    rpc: "https://rpc.ankr.com/polygon",
    key: getEnvVar("ETH_KEY"),
  },
  Avalanche: {
    rpc: "https://rpc.ankr.com/avalanche",
    key: getEnvVar("ETH_KEY"),
  },
  Algorand: {
    rpc: "https://mainnet-api.algonode.cloud",
    key: getEnvVar("ALGORAND_KEY"),
  },
  Klaytn: {
    rpc: "https://public-en.node.kaia.io",
    key: getEnvVar("ETH_KEY"),
  },
  Celo: {
    rpc: "https://forno.celo.org",
    key: getEnvVar("ETH_KEY"),
  },
  Near: {
    rpc: "https://rpc.mainnet.near.org",
    key: getEnvVar("NEAR_KEY"),
    networkId: "mainnet",
  },
  Injective: {
    rpc: "http://sentry0.injective.network:26657",
    key: getEnvVar("INJECTIVE_KEY"),
  },
  Aptos: {
    rpc: "https://fullnode.mainnet.aptoslabs.com/v1",
    key: getEnvVar("APTOS_KEY"),
  },
  Sui: {
    // used as the gRPC base URL: Sui deprecated JSON-RPC on public fullnodes
    rpc: "https://fullnode.mainnet.sui.io:443",
    key: getEnvVar("SUI_KEY"),
  },
  Pythnet: {
    rpc: "http://api.pythnet.pyth.network:8899/",
    key: getEnvVar("SOLANA_KEY"),
  },
  Arbitrum: {
    rpc: "https://arb1.arbitrum.io/rpc",
    key: getEnvVar("ETH_KEY"),
  },
  Optimism: {
    rpc: "https://mainnet.optimism.io",
    key: getEnvVar("ETH_KEY"),
  },
  Base: {
    rpc: "https://mainnet.base.org",
    key: getEnvVar("ETH_KEY"),
  },
  Sei: {
    rpc: "https://sei-rpc.polkachu.com/",
    key: getEnvVar("SEI_KEY"),
  },
  Fogo: {
    rpc: "https://mainnet.fogo.io",
    key: getEnvVar("SOLANA_KEY"),
  },
};

const Testnet = {
  ...defaultConnections("Testnet"),
  Solana: {
    rpc: "https://api.devnet.solana.com",
    key: getEnvVar("SOLANA_KEY_TESTNET"),
  },
  Ethereum: {
    rpc: `https://rpc.ankr.com/eth_goerli`,
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Bsc: {
    rpc: "https://data-seed-prebsc-1-s1.binance.org:8545",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Polygon: {
    rpc: `https://rpc.ankr.com/polygon_mumbai`,
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Avalanche: {
    rpc: "https://rpc.ankr.com/avalanche_fuji",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Algorand: {
    rpc: "https://testnet-api.algonode.cloud",
    key: getEnvVar("ALGORAND_KEY_TESTNET"),
  },
  Klaytn: {
    rpc: "https://public-en-kairos.node.kaia.io",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Celo: {
    rpc: "https://alfajores-forno.celo-testnet.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Near: {
    rpc: "https://rpc.testnet.near.org",
    key: getEnvVar("NEAR_KEY_TESTNET"),
    networkId: "testnet",
  },
  Injective: {
    rpc: "https://k8s.testnet.tm.injective.network:443",
    key: getEnvVar("INJECTIVE_KEY_TESTNET"),
  },
  Osmosis: {
    rpc: "https://rpc.testnet.osmosis.zone",
    key: getEnvVar("OSMOSIS_KEY_TESTNET"),
  },
  Aptos: {
    rpc: "https://fullnode.testnet.aptoslabs.com/v1",
    key: getEnvVar("APTOS_TESTNET"),
  },
  Sui: {
    // used as the gRPC base URL: Sui deprecated JSON-RPC on public fullnodes
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
  Linea: {
    rpc: "https://rpc.sepolia.linea.build",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Berachain: {
    rpc: "https://bepolia.rpc.berachain.com/",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Seievm: {
    rpc: "https://evm-rpc-testnet.sei-apis.com/",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Sepolia: {
    rpc: "https://rpc.ankr.com/eth_sepolia",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Holesky: {
    rpc: "https://rpc.ankr.com/eth_holesky",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Moonbeam: {
    rpc: "https://rpc.api.moonbase.moonbeam.network",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Arbitrum: {
    rpc: "https://goerli-rollup.arbitrum.io/rpc",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Optimism: {
    rpc: "https://goerli.optimism.io",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Base: {
    rpc: "https://goerli.base.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  ArbitrumSepolia: {
    rpc: "https://arbitrum-sepolia.publicnode.com",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  BaseSepolia: {
    rpc: "https://sepolia.base.org",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  OptimismSepolia: {
    rpc: "https://rpc.ankr.com/optimism_sepolia",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  PolygonSepolia: {
    rpc: "https://rpc-amoy.polygon.technology/",
    key: getEnvVar("ETH_KEY_TESTNET"),
  },
  Fogo: {
    rpc: "https://testnet.fogo.io",
    key: getEnvVar("SOLANA_KEY_TESTNET"),
  },
};

const Devnet = {
  ...defaultConnections("Devnet"),
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
  Wormchain: {
    rpc: "http://localhost:1319",
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
