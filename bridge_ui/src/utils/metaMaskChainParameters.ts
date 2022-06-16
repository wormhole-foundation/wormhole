// https://docs.metamask.io/guide/rpc-api.html#wallet-addethereumchain as per EIP-3085
export interface AddEthereumChainParameter {
  chainId: string; // A 0x-prefixed hexadecimal string
  chainName: string;
  nativeCurrency: {
    name: string;
    symbol: string; // 2-6 characters long
    decimals: 18;
  };
  rpcUrls: string[];
  blockExplorerUrls?: string[];
  iconUrls?: string[]; // Currently ignored.
}

// https://chainid.network/chains.json for chain info
export const METAMASK_CHAIN_PARAMETERS: {
  [evmChainId: number]: AddEthereumChainParameter;
} = {
  1: {
    chainId: "0x1",
    chainName: "Ethereum Mainnet",
    nativeCurrency: { name: "Ether", symbol: "ETH", decimals: 18 },
    rpcUrls: ["https://rpc.ankr.com/eth"],
    blockExplorerUrls: ["https://etherscan.io"],
  },
  3: {
    chainId: "0x3",
    chainName: "Ropsten",
    nativeCurrency: { name: "Ropsten Ether", symbol: "ROP", decimals: 18 },
    rpcUrls: ["https://rpc.ankr.com/eth_ropsten"],
    blockExplorerUrls: ["https://ropsten.etherscan.io"],
  },
  5: {
    chainId: "0x5",
    chainName: "Görli",
    nativeCurrency: { name: "Görli Ether", symbol: "GOR", decimals: 18 },
    rpcUrls: ["https://rpc.ankr.com/eth_goerli"],
    blockExplorerUrls: ["https://goerli.etherscan.io"],
  },
  56: {
    chainId: "0x38",
    chainName: "Binance Smart Chain Mainnet",
    nativeCurrency: {
      name: "Binance Chain Native Token",
      symbol: "BNB",
      decimals: 18,
    },
    rpcUrls: ["https://bsc-dataseed.binance.org"],
    blockExplorerUrls: ["https://bscscan.com"],
  },
  97: {
    chainId: "0x61",
    chainName: "Binance Smart Chain Testnet",
    nativeCurrency: {
      name: "Binance Chain Native Token",
      symbol: "BNB",
      decimals: 18,
    },
    rpcUrls: ["https://data-seed-prebsc-1-s1.binance.org:8545"],
    blockExplorerUrls: ["https://testnet.bscscan.com"],
  },
  137: {
    chainId: "0x89",
    chainName: "Polygon Mainnet",
    nativeCurrency: { name: "MATIC", symbol: "MATIC", decimals: 18 },
    rpcUrls: ["https://polygon-rpc.com"],
    blockExplorerUrls: ["https://polygonscan.com"],
  },
  250: {
    chainId: "0xfa",
    chainName: "Fantom Opera",
    nativeCurrency: { name: "Fantom", symbol: "FTM", decimals: 18 },
    rpcUrls: ["https://rpc.ftm.tools"],
    blockExplorerUrls: ["https://ftmscan.com"],
  },
  596: {
    chainId: "0x254",
    chainName: "Karura Testnet",
    nativeCurrency: { name: "Karura Token", symbol: "KAR", decimals: 18 },
    rpcUrls: ["https://karura-dev.aca-dev.network/eth/http"],
    blockExplorerUrls: ["https://blockscout.karura-dev.aca-dev.network"],
  },
  597: {
    chainId: "0x255",
    chainName: "Acala Testnet",
    nativeCurrency: { name: "Acala Token", symbol: "ACA", decimals: 18 },
    rpcUrls: ["https://acala-dev.aca-dev.network/eth/http"],
    blockExplorerUrls: ["https://blockscout.acala-dev.aca-dev.network"],
  },
  686: {
    chainId: "0x2AE",
    chainName: "Karura",
    nativeCurrency: { name: "Karura Token", symbol: "KAR", decimals: 18 },
    rpcUrls: ["https://eth-rpc-karura.aca-api.network"],
    blockExplorerUrls: ["https://blockscout.karura.network"],
  },
  787: {
    chainId: "0x313",
    chainName: "Acala",
    nativeCurrency: { name: "Acala Token", symbol: "ACA", decimals: 18 },
    rpcUrls: ["https://eth-rpc-acala.aca-api.network"],
    blockExplorerUrls: ["https://blockscout.acala.network"],
  },
  4002: {
    chainId: "0xfa2",
    chainName: "Fantom Testnet",
    nativeCurrency: { name: "Fantom", symbol: "FTM", decimals: 18 },
    rpcUrls: ["https://rpc.testnet.fantom.network"],
    blockExplorerUrls: ["https://testnet.ftmscan.com"],
  },
  8217: {
    chainId: "0x2019",
    chainName: "Klaytn",
    nativeCurrency: { name: "Klay", symbol: "KLAY", decimals: 18 },
    rpcUrls: ["https://public-node-api.klaytnapi.com/v1/cypress"],
    blockExplorerUrls: ["https://scope.klaytn.com"],
  },
  1001: {
    chainId: "0x3E9",
    chainName: "Klaytn Testnet Baobab",
    nativeCurrency: { name: "Klay", symbol: "KLAY", decimals: 18 },
    rpcUrls: ["https://api.baobab.klaytn.net:8651"],
    blockExplorerUrls: ["https://baobab.scope.klaytn.com/"],
  },
  42220: {
    chainId: "0xa4ec",
    chainName: "Celo",
    nativeCurrency: { name: "Celo", symbol: "CELO", decimals: 18 },
    rpcUrls: ["https://forno.celo.org"],
    blockExplorerUrls: ["https://explorer.celo.org"],
  },
  44787: {
    chainId: "0xaef3",
    chainName: "Celo (Alfajores Testnet)",
    nativeCurrency: { name: "Celo", symbol: "CELO", decimals: 18 },
    rpcUrls: ["https://alfajores-forno.celo-testnet.org"],
    blockExplorerUrls: ["https://alfajores-blockscout.celo-testnet.org"],
  },
  42261: {
    chainId: "0xa515",
    chainName: "Emerald Paratime Testnet",
    nativeCurrency: { name: "Emerald Rose", symbol: "ROSE", decimals: 18 },
    rpcUrls: ["https://testnet.emerald.oasis.dev"],
    blockExplorerUrls: ["https://testnet.explorer.emerald.oasis.dev"],
  },
  42262: {
    chainId: "0xa516",
    chainName: "Emerald Paratime Mainnet",
    nativeCurrency: { name: "Emerald Rose", symbol: "ROSE", decimals: 18 },
    rpcUrls: ["https://emerald.oasis.dev"],
    blockExplorerUrls: ["https://explorer.emerald.oasis.dev"],
  },
  43113: {
    chainId: "0xa869",
    chainName: "Avalanche Fuji Testnet",
    nativeCurrency: { name: "Avalanche", symbol: "AVAX", decimals: 18 },
    rpcUrls: ["https://api.avax-test.network/ext/bc/C/rpc"],
    blockExplorerUrls: ["https://testnet.snowtrace.io"],
  },
  43114: {
    chainId: "0xa86a",
    chainName: "Avalanche C-Chain",
    nativeCurrency: { name: "Avalanche", symbol: "AVAX", decimals: 18 },
    rpcUrls: ["https://api.avax.network/ext/bc/C/rpc"],
    blockExplorerUrls: ["https://snowtrace.io"],
  },
  80001: {
    chainId: "0x13881",
    chainName: "Mumbai",
    nativeCurrency: { name: "MATIC", symbol: "MATIC", decimals: 18 },
    rpcUrls: ["https://rpc-mumbai.maticvigil.com"],
    blockExplorerUrls: ["https://mumbai.polygonscan.com"],
  },
  245022926: {
    chainId: "0xE9AC0CE",
    chainName: "remote proxy — solana devnet",
    nativeCurrency: { name: "NEON", symbol: "NEON", decimals: 18 },
    rpcUrls: ["https://proxy.devnet.neonlabs.org/solana"],
    blockExplorerUrls: ["https://neonscan.org/"],
  },
  1313161554: {
    chainId: "0x4e454152",
    chainName: "Aurora Mainnet",
    nativeCurrency: { name: "Ether", symbol: "ETH", decimals: 18 },
    rpcUrls: ["https://mainnet.aurora.dev"],
    blockExplorerUrls: ["https://aurorascan.dev"],
  },
  1313161555: {
    chainId: "0x4e454153",
    chainName: "Aurora Testnet",
    nativeCurrency: { name: "Ether", symbol: "ETH", decimals: 18 },
    rpcUrls: ["https://testnet.aurora.dev"],
    blockExplorerUrls: ["https://testnet.aurorascan.dev"],
  },
};

export interface EvmRpcMap {
  [chainId: string]: string;
}

export const EVM_RPC_MAP = Object.entries(METAMASK_CHAIN_PARAMETERS).reduce(
  (evmRpcMap, [evmChainId, { rpcUrls }]) => {
    if (rpcUrls.length > 0) {
      evmRpcMap[evmChainId] = rpcUrls[0];
    }
    return evmRpcMap;
  },
  {} as EvmRpcMap
);
