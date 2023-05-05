import { ChainId, Network, ChainName } from "../";
import { ethers } from "ethers";
import { CoreRelayer__factory, CoreRelayer } from "../ethers-contracts/";

const TESTNET = [
  {
    chainId: 4,
    coreRelayerAddress: "0x6Bf598B0eb6aef9B163565763Fe50e54d230eD4E",
  },
  {
    chainId: 5,
    coreRelayerAddress: "0x0c97Ef9C224b7EB0BA5e4A9fd2740EC3AeAfc9c3",
  },
  {
    chainId: 6,
    coreRelayerAddress: "0xf4e844a9B75BB532e67E654F7F80C6232e5Ea7a0",
  },
  {
    chainId: 14,
    coreRelayerAddress: "0xF08B7c0CFf448174a7007CF5f12023C72C0e84f0",
  },
  {
    chainId: 16,
    coreRelayerAddress: "0xd20d484eC6c57448d6871F91F4527260FD4aC141",
  },
];

const DEVNET = [
  {
    chainId: 2,
    coreRelayerAddress: "0x34dc8837Fb909b83CDe5479015A9925acFb63165",
  },
  {
    chainId: 4,
    coreRelayerAddress: "0x34dc8837Fb909b83CDe5479015A9925acFb63165",
  },
];

const MAINNET: any[] = [];

export function getWormholeRelayerAddress(
  chainId: ChainId,
  env: Network
): string {
  if (env == "TESTNET") {
    const address = TESTNET.find(
      (x) => x.chainId == chainId
    )?.coreRelayerAddress;
    if (!address) {
      throw Error("Invalid chain ID");
    }
    return address;
  } else if (env == "MAINNET") {
    const address = MAINNET.find(
      (x) => x.chainId == chainId
    )?.coreRelayerAddress;
    if (!address) {
      throw Error("Invalid chain ID");
    }
    return address;
  } else if (env == "DEVNET") {
    const address = DEVNET.find(
      (x) => x.chainId == chainId
    )?.coreRelayerAddress;
    if (!address) {
      throw Error("Invalid chain ID");
    }
    return address;
  } else {
    throw Error("Invalid environment");
  }
}

export function getWormholeRelayer(
  chainId: ChainId,
  env: Network,
  provider: ethers.providers.Provider | ethers.Signer,
  wormholeRelayerAddress?: string
): CoreRelayer {
  const thisChainsRelayer = wormholeRelayerAddress || getWormholeRelayerAddress(chainId, env);
  const contract = CoreRelayer__factory.connect(thisChainsRelayer, provider);
  return contract;
}

export const RPCS_BY_CHAIN: {
  [key in Network]: { [key in ChainName]?: string };
} = {
  MAINNET: {
    ethereum: process.env.ETH_RPC,
    bsc: process.env.BSC_RPC || "https://bsc-dataseed2.defibit.io",
    polygon: "https://rpc.ankr.com/polygon",
    avalanche: "https://rpc.ankr.com/avalanche",
    oasis: "https://emerald.oasis.dev",
    algorand: "https://mainnet-api.algonode.cloud",
    fantom: "https://rpc.ankr.com/fantom",
    karura: "https://eth-rpc-karura.aca-api.network",
    acala: "https://eth-rpc-acala.aca-api.network",
    klaytn: "https://klaytn-mainnet-rpc.allthatnode.com:8551",
    celo: "https://forno.celo.org",
    moonbeam: "https://rpc.ankr.com/moonbeam",
    arbitrum: "https://rpc.ankr.com/arbitrum",
    optimism: "https://rpc.ankr.com/optimism",
    aptos: "https://fullnode.mainnet.aptoslabs.com/",
    near: "https://rpc.mainnet.near.org",
    xpla: "https://dimension-lcd.xpla.dev",
    terra2: "https://phoenix-lcd.terra.dev",
    terra: "https://columbus-fcd.terra.dev",
    injective: "https://k8s.mainnet.lcd.injective.network",
    solana: process.env.SOLANA_RPC ?? "https://api.mainnet-beta.solana.com",
  },
  TESTNET: {
    solana: "https://api.devnet.solana.com",
    terra: "https://bombay-lcd.terra.dev",
    ethereum: "https://rpc.ankr.com/eth_goerli",
    bsc: "https://data-seed-prebsc-1-s1.binance.org:8545",
    polygon: "https://rpc.ankr.com/polygon_mumbai",
    avalanche: "https://rpc.ankr.com/avalanche_fuji",
    oasis: "https://testnet.emerald.oasis.dev",
    algorand: "https://testnet-api.algonode.cloud",
    fantom: "https://rpc.testnet.fantom.network",
    aurora: "https://testnet.aurora.dev",
    karura: "https://karura-dev.aca-dev.network/eth/http",
    acala: "https://acala-dev.aca-dev.network/eth/http",
    klaytn: "https://api.baobab.klaytn.net:8651",
    celo: "https://alfajores-forno.celo-testnet.org",
    near: "https://rpc.testnet.near.org",
    injective: "https://k8s.testnet.tm.injective.network:443",
    aptos: "https://fullnode.testnet.aptoslabs.com/v1",
    pythnet: "https://api.pythtest.pyth.network/",
    xpla: "https://cube-lcd.xpla.dev:443",
    moonbeam: "https://rpc.api.moonbase.moonbeam.network",
    neon: "https://proxy.devnet.neonlabs.org/solana",
    terra2: "https://pisco-lcd.terra.dev",
    arbitrum: "https://goerli-rollup.arbitrum.io/rpc",
    optimism: "https://goerli.optimism.io",
    gnosis: "https://sokol.poa.network/",
  },
  DEVNET: {
    ethereum: "http://localhost:8545",
    bsc: "http://localhost:8546",
  },
};

export const GUARDIAN_RPC_HOSTS = [
  "https://wormhole-v2-mainnet-api.certus.one",
  "https://wormhole.inotel.ro",
  "https://wormhole-v2-mainnet-api.mcf.rocks",
  "https://wormhole-v2-mainnet-api.chainlayer.network",
  "https://wormhole-v2-mainnet-api.staking.fund",
];
