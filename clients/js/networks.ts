import { ChainName } from "../../sdk/js/src/utils/consts"

require("dotenv").config({ path: `${process.env.HOME}/.wormhole/.env` })

function get_env_var(env: string): string {
  const v = process.env[env]
  if (v === undefined) {
    throw new Error(`Env variable ${env} undefined`)
  }
  return v
}

export type Connection = {
  rpc: string | undefined,
  key: string | undefined,
}

export type ChainConnections = {
  [chain in ChainName]: Connection
}

const MAINNET = {
  unset: {
    rpc: undefined,
    key: undefined
  },
  solana: {
    rpc: 'https://api.mainnet-beta.solana.com',
    key: get_env_var("SOLANA_KEY")
  },
  terra: {
    rpc: "https://lcd.terra.dev",
    chain_id: "columbus-5",
    key: get_env_var("TERRA_MNEMONIC")
  },
  ethereum: {
    rpc: `https://mainnet.infura.io/v3/${get_env_var("INFURA_KEY")}`,
    key: get_env_var("ETH_KEY")
  },
  bsc: {
    rpc: "https://bsc-dataseed.binance.org/",
    key: get_env_var("ETH_KEY")
  },
  polygon: {
    rpc: "https://polygon-rpc.com",
    key: get_env_var("ETH_KEY")
  },
  avalanche: {
    rpc: "https://api.avax.network/ext/bc/C/rpc",
    key: get_env_var("ETH_KEY"),
  },
  algorand: {
    rpc: undefined,
    key: undefined
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
    rpc: undefined,
    key: get_env_var("ETH_KEY"),
  },
  acala: {
    rpc: undefined,
    key: get_env_var("ETH_KEY"),
  },
  klaytn: {
    rpc: undefined,
    key: get_env_var("ETH_KEY"),
  },
  ropsten: {
    rpc: `https://ropsten.infura.io/v3/${get_env_var("INFURA_KEY")}`,
    key: get_env_var("ETH_KEY")
  },
}

const TESTNET = {
  unset: {
    rpc: undefined,
    key: undefined
  },
  solana: {
    rpc: 'https://api.devnet.solana.com',
    key: get_env_var("SOLANA_KEY")
  },
  terra: {
    rpc: "https://bombay-lcd.terra.dev",
    chain_id: "bombay-12",
    key: get_env_var("TERRA_MNEMONIC")
  },
  ethereum: {
    rpc: `https://goerli.infura.io/v3/${get_env_var("INFURA_KEY")}`,
    key: get_env_var("ETH_KEY")
  },
  bsc: {
    rpc: "https://data-seed-prebsc-1-s1.binance.org:8545",
    key: get_env_var("ETH_KEY")
  },
  polygon: {
    rpc: `https://polygon-mumbai.infura.io/v3/${get_env_var("INFURA_KEY")}`,
    key: get_env_var("ETH_KEY")
  },
  avalanche: {
    rpc: "https://api.avax-test.network/ext/bc/C/rpc",
    key: get_env_var("ETH_KEY"),
  },
  oasis: {
    rpc: "https://testnet.emerald.oasis.dev",
    key: get_env_var("ETH_KEY"),
  },
  algorand: {
    rpc: undefined,
    key: undefined
  },
  fantom: {
    rpc: "https://rpc.testnet.fantom.network",
    key: get_env_var("ETH_KEY"),
  },
  aurora: {
    rpc: "https://testnet.aurora.dev",
    key: get_env_var("ETH_KEY"),
  },
  karura: {
    rpc: "http://103.253.145.222:8545",
    key: get_env_var("ETH_KEY"),
  },
  acala: {
    rpc: "http://157.245.252.103:8545",
    key: get_env_var("ETH_KEY"),
  },
  klaytn: {
    rpc: "https://api.baobab.klaytn.net:8651",
    key: get_env_var("ETH_KEY"),
  },
  ropsten: {
    rpc: `https://ropsten.infura.io/v3/${get_env_var("INFURA_KEY")}`,
    key: get_env_var("ETH_KEY")
  },
}

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
const isTestnetConnections: ChainConnections = TESTNET

/**
 *
 * See [[isTestnetContracts]]
 */
const isMainnetConnections: ChainConnections = MAINNET

export const NETWORKS = { MAINNET, TESTNET }
