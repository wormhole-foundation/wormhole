import { ChainId, Network, ChainName, CHAIN_ID_TO_NAME,} from "../";
import { ethers } from "ethers";
import { WormholeRelayer__factory, WormholeRelayer } from "../ethers-contracts/";

type AddressInfo = {wormholeRelayerAddress?: string, mockDeliveryProviderAddress?: string, mockIntegrationAddress?: string}

const TESTNET: {[K in ChainName]?: AddressInfo} = {
  bsc: {
    wormholeRelayerAddress: "0x6Bf598B0eb6aef9B163565763Fe50e54d230eD4E",
  },
  polygon: {
    wormholeRelayerAddress: "0x0c97Ef9C224b7EB0BA5e4A9fd2740EC3AeAfc9c3",
  },
  avalanche: {
    wormholeRelayerAddress: "0xf4e844a9B75BB532e67E654F7F80C6232e5Ea7a0",
  },
  celo: {
    wormholeRelayerAddress: "0xF08B7c0CFf448174a7007CF5f12023C72C0e84f0",
  },
  moonbeam: {
    wormholeRelayerAddress: "0xd20d484eC6c57448d6871F91F4527260FD4aC141",
  },
}

const DEVNET: {[K in ChainName]?: AddressInfo} = {
  ethereum: {
    wormholeRelayerAddress: "0xE66C1Bc1b369EF4F376b84373E3Aa004E8F4C083",
    mockDeliveryProviderAddress: "0x1ef9e15c3bbf0555860b5009B51722027134d53a",
    mockIntegrationAddress: "0x0eb0dD3aa41bD15C706BC09bC03C002b7B85aeAC",
  },
  bsc: {
    wormholeRelayerAddress: "0xE66C1Bc1b369EF4F376b84373E3Aa004E8F4C083",
    mockDeliveryProviderAddress: "0x1ef9e15c3bbf0555860b5009B51722027134d53a",
    mockIntegrationAddress: "0x0eb0dD3aa41bD15C706BC09bC03C002b7B85aeAC",
  },
};

const MAINNET: {[K in ChainName]?: AddressInfo} = {};

export const RELAYER_CONTRACTS = {MAINNET, TESTNET, DEVNET};

export function getAddressInfo(chainName: ChainName, env: Network): AddressInfo {
  const result: AddressInfo | undefined = RELAYER_CONTRACTS[env][chainName];
  if(!result) throw Error(`No address info for chain ${chainName} on ${env}`);
  return result;
}

export function getWormholeRelayerAddress(
  chainName: ChainName,
  env: Network
): string {
  const result = getAddressInfo(chainName, env).wormholeRelayerAddress;
  if(!result) throw Error(`No Wormhole Relayer Address for chain ${chainName}, network ${env}`);
  return result;
}

export function getWormholeRelayer(
  chainName: ChainName,
  env: Network,
  provider: ethers.providers.Provider | ethers.Signer,
  wormholeRelayerAddress?: string
): WormholeRelayer {
  const thisChainsRelayer = wormholeRelayerAddress || getWormholeRelayerAddress(chainName, env);
  const contract = WormholeRelayer__factory.connect(thisChainsRelayer, provider);
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
    sui: "https://fullnode.mainnet.sui.io:443",
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
