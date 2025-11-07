// Constants for Stacks ↔ Ethereum integration testing
// Based on existing devnet configuration

// Stacks Configuration (from existing test files)
export const STACKS_API_URL = "http://stacks-node:20443";
export const STACKS_PRIVATE_KEY =
  "714a5bf161a680ebb2670c5ea6e8bcd75f299eae234412af0cf12d21e11ae09901";

// Chain IDs (Wormhole numbering system)
export const CHAIN_ID_STACKS = 60;
export const CHAIN_ID_ETHEREUM = 2;

// Guardian Spy Service Host
export const SPY_SERVICE_HOST = "spy:7072";

// Ethereum Devnet Contract Addresses (from research)
export const ETHEREUM_CONTRACTS = {
  WORMHOLE_CORE: "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550",
  TOKEN_BRIDGE: "0x0290FB167208Af455bB137780163b7B7a9a10C16",
  TEST_TOKEN: "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A",
} as const;

// Ethereum Configuration
export const ETH_NODE_URL = "http://localhost:8545";

// Ethereum Private Keys (from existing SDK test configuration)
export const ETH_PRIVATE_KEY =
  "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"; // account 0
export const ETH_PRIVATE_KEY_STACKS =
  "0x6370fd033278c143179d81c5526140625662b8daa446c22ee2d73db3707e620c"; // account 1 - for Stacks tests

// Stacks Contract Names (based on available contracts)
export const STACKS_CONTRACTS = {
  CORE_STATE: "wormhole-core-state",
  CORE_V4: "wormhole-core-v4",
  CORE_PROXY: "wormhole-core-proxy-v2",
} as const;

// Ethereum NTT Contract Addresses (deployed by 05-deploy-ntt-ethereum.ts)
export const ETHEREUM_NTT_CONTRACTS = {
  DUMMY_TOKEN: "0xfE82e8f24A51E670133f4268cDfc164c49FC3b37",
  TRANSCEIVER_STRUCTS_LIBRARY: "0xb4fFe5983B0B748124577Af4d16953bd096b6897",
  NTT_MANAGER: "0x6f84742680311CEF5ba42bc10A71a4708b4561d1",
  WORMHOLE_TRANSCEIVER: "0x25AF99b922857C37282f578F428CB7f34335B379",
} as const;

// NTT Configuration Constants
export const NTT_CONFIG = {
  MODE: 0, // LOCKING mode - preserve STX supply on Stacks
  RATE_LIMIT_DURATION: 86400, // 24 hours in seconds
  CONSISTENCY_LEVEL: 202, // Finalized
  GAS_LIMIT: 500000,
  OUTBOUND_LIMIT: "1000000000000000000000000", // 1M tokens (scaled by 18 decimals)
  THRESHOLD: 1, // Single transceiver setup
} as const;
