import { ethers } from "ethers";
import { ChainName, Network } from "../";
import {
  WormholeRelayer,
  WormholeRelayer__factory,
} from "../ethers-relayer-contracts/";

type AddressInfo = {
  wormholeRelayerAddress?: string;
  mockDeliveryProviderAddress?: string;
  mockIntegrationAddress?: string;
};

const TESTNET: { [K in ChainName]?: AddressInfo } = {
  ethereum: {
    wormholeRelayerAddress: "0x28D8F1Be96f97C1387e94A53e00eCcFb4E75175a",
    mockDeliveryProviderAddress: "0xD1463B4fe86166768d2ff51B1A928beBB5c9f375",
    mockIntegrationAddress: "0xb81bc199b73AB34c393a4192C163252116a03370",
  },
  bsc: {
    wormholeRelayerAddress: "0x80aC94316391752A193C1c47E27D382b507c93F3",
    mockDeliveryProviderAddress: "0x60a86b97a7596eBFd25fb769053894ed0D9A8366",
    mockIntegrationAddress: "0xb6A04D6672F005787147472Be20d39741929Aa03",
  },
  polygon: {
    wormholeRelayerAddress: "0x0591C25ebd0580E0d4F27A82Fc2e24E7489CB5e0",
    mockDeliveryProviderAddress: "0x60a86b97a7596eBFd25fb769053894ed0D9A8366",
    mockIntegrationAddress: "0x3bF0c43d88541BBCF92bE508ec41e540FbF28C56",
  },
  avalanche: {
    wormholeRelayerAddress: "0xA3cF45939bD6260bcFe3D66bc73d60f19e49a8BB",
    mockDeliveryProviderAddress: "0x60a86b97a7596eBFd25fb769053894ed0D9A8366",
    mockIntegrationAddress: "0x5E52f3eB0774E5e5f37760BD3Fca64951D8F74Ae",
  },
  celo: {
    wormholeRelayerAddress: "0x306B68267Deb7c5DfCDa3619E22E9Ca39C374f84",
    mockDeliveryProviderAddress: "0x60a86b97a7596eBFd25fb769053894ed0D9A8366",
    mockIntegrationAddress: "0x7f1d8E809aBB3F6Dc9B90F0131C3E8308046E190",
  },
  moonbeam: {
    wormholeRelayerAddress: "0x0591C25ebd0580E0d4F27A82Fc2e24E7489CB5e0",
    mockDeliveryProviderAddress: "0x60a86b97a7596eBFd25fb769053894ed0D9A8366",
    mockIntegrationAddress: "0x3bF0c43d88541BBCF92bE508ec41e540FbF28C56",
  },
  arbitrum: {
    wormholeRelayerAddress: "0xAd753479354283eEE1b86c9470c84D42f229FF43",
    mockDeliveryProviderAddress: "0x90995DBd1aae85872451b50A569dE947D34ac4ee",
    mockIntegrationAddress: "0x0de48f34E14d08934DA1eA2286Be1b2BED5c062a",
  },
  optimism: {
    wormholeRelayerAddress: "0x01A957A525a5b7A72808bA9D10c389674E459891",
    mockDeliveryProviderAddress: "0xfCe1Df3EF22fe5Cb7e2f5988b7d58fF633a313a7",
    mockIntegrationAddress: "0x421e0bb71dDeeC727Af79766423d33D8FD7dB963",
  },
  base: {
    wormholeRelayerAddress: "0xea8029CD7FCAEFFcD1F53686430Db0Fc8ed384E1",
    mockDeliveryProviderAddress: "0x60a86b97a7596eBFd25fb769053894ed0D9A8366",
    mockIntegrationAddress: "0x9Ee656203B0DC40cc1bA3f4738527779220e3998",
  },
  sepolia: {
    wormholeRelayerAddress: "0x7B1bD7a6b4E61c2a123AC6BC2cbfC614437D0470",
    mockDeliveryProviderAddress: "0x7A0a53847776f7e94Cc35742971aCb2217b0Db81",
    mockIntegrationAddress: "0x68b7Cd0d27a6F04b2F65e11DD06182EFb255c9f0",
  },
  arbitrum_sepolia: {
    wormholeRelayerAddress: "0x7B1bD7a6b4E61c2a123AC6BC2cbfC614437D0470",
    mockDeliveryProviderAddress: "0x7A0a53847776f7e94Cc35742971aCb2217b0Db81",
    mockIntegrationAddress: "0x2B1502Ffe717817A0A101a687286bE294fe495f7",
  },
  optimism_sepolia: {
    wormholeRelayerAddress: "0x93BAD53DDfB6132b0aC8E37f6029163E63372cEE",
    mockDeliveryProviderAddress: "0x7A0a53847776f7e94Cc35742971aCb2217b0Db81",
    mockIntegrationAddress: "0xA404B69582bac287a7455FFf315938CCd92099c1",
  },
  base_sepolia: {
    wormholeRelayerAddress: "0x93BAD53DDfB6132b0aC8E37f6029163E63372cEE",
    mockDeliveryProviderAddress: "0x7A0a53847776f7e94Cc35742971aCb2217b0Db81",
    mockIntegrationAddress: "0xA404B69582bac287a7455FFf315938CCd92099c1",
  },
};

const DEVNET: { [K in ChainName]?: AddressInfo } = {
  ethereum: {
    wormholeRelayerAddress: "0xb98F46E96cb1F519C333FdFB5CCe0B13E0300ED4",
    mockDeliveryProviderAddress: "0x1ef9e15c3bbf0555860b5009B51722027134d53a",
    mockIntegrationAddress: "0x0eb0dD3aa41bD15C706BC09bC03C002b7B85aeAC",
  },
  bsc: {
    wormholeRelayerAddress: "0xb98F46E96cb1F519C333FdFB5CCe0B13E0300ED4",
    mockDeliveryProviderAddress: "0x1ef9e15c3bbf0555860b5009B51722027134d53a",
    mockIntegrationAddress: "0x0eb0dD3aa41bD15C706BC09bC03C002b7B85aeAC",
  },
};

const MAINNET: { [K in ChainName]?: AddressInfo } = {
  ethereum: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  bsc: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  polygon: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  avalanche: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  fantom: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  klaytn: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  celo: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  acala: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  karura: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  moonbeam: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  arbitrum: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  optimism: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  base: {
    wormholeRelayerAddress: "0x706f82e9bb5b0813501714ab5974216704980e31",
  },
  scroll: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
  blast: {
    wormholeRelayerAddress: "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911",
  },
};

export const RELAYER_CONTRACTS = { MAINNET, TESTNET, DEVNET };

export function getAddressInfo(
  chainName: ChainName,
  env: Network
): AddressInfo {
  const result: AddressInfo | undefined = RELAYER_CONTRACTS[env][chainName];
  if (!result) throw Error(`No address info for chain ${chainName} on ${env}`);
  return result;
}

export function getWormholeRelayerAddress(
  chainName: ChainName,
  env: Network
): string {
  const result = getAddressInfo(chainName, env).wormholeRelayerAddress;
  if (!result)
    throw Error(
      `No Wormhole Relayer Address for chain ${chainName}, network ${env}`
    );
  return result;
}

export function getWormholeRelayer(
  chainName: ChainName,
  env: Network,
  provider: ethers.providers.Provider | ethers.Signer,
  wormholeRelayerAddress?: string
): WormholeRelayer {
  const thisChainsRelayer =
    wormholeRelayerAddress || getWormholeRelayerAddress(chainName, env);
  const contract = WormholeRelayer__factory.connect(
    thisChainsRelayer,
    provider
  );
  return contract;
}

export const RPCS_BY_CHAIN: {
  [key in Network]: { [key in ChainName]?: string };
} = {
  MAINNET: {
    ethereum: "https://rpc.ankr.com/eth",
    bsc: "https://bsc-dataseed2.defibit.io",
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
    solana: "https://api.mainnet-beta.solana.com",
    base: "https://mainnet.base.org",
  },
  TESTNET: {
    solana: "https://api.devnet.solana.com",
    terra: "https://bombay-lcd.terra.dev",
    ethereum: "https://rpc.ankr.com/eth_goerli",
    bsc: "https://bsc-testnet.publicnode.com",
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
    rootstock: "https://public-node.rsk.co",
    base: "https://goerli.base.org",
    sepolia: "https://rpc.ankr.com/eth_sepolia",
    arbitrum_sepolia: "https://sepolia-rollup.arbitrum.io/rpc",
    optimism_sepolia: "https://sepolia.optimism.io",
    base_sepolia: "https://sepolia.base.org",
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

export const getCircleAPI = (environment: Network) => {
  return environment === "TESTNET"
    ? "https://iris-api-sandbox.circle.com/v1/attestations/"
    : "https://iris-api.circle.com/v1/attestations/";
};

export const getWormscanAPI = (_network: Network) => {
  switch (_network) {
    case "MAINNET":
      return "https://api.wormholescan.io/";
    case "TESTNET":
      return "https://api.testnet.wormholescan.io/";
    default:
      // possible extension for tilt/ci - search through the guardian api
      // at localhost:7071 (tilt) or guardian:7071 (ci)
      throw new Error("Not testnet or mainnet - so no wormscan api access");
  }
};

export const getNameFromCCTPDomain = (
  domain: number,
  environment: Network = "MAINNET"
): ChainName | undefined => {
  if (domain === 0) return environment === "MAINNET" ? "ethereum" : "sepolia";
  else if (domain === 1) "avalanche";
  else if (domain === 2)
    return environment === "MAINNET" ? "optimism" : "optimism_sepolia";
  else if (domain === 3)
    return environment === "MAINNET" ? "arbitrum" : "arbitrum_sepolia";
  else if (domain === 6)
    return environment === "MAINNET" ? "base" : "base_sepolia";
  else return undefined;
};
