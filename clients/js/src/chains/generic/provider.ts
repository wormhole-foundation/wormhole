import { CosmWasmClient } from "@cosmjs/cosmwasm-stargate";
import {
  Network as InjectiveNetwork,
  getNetworkEndpoints,
} from "@injectivelabs/networks";
import { ChainGrpcWasmApi } from "@injectivelabs/sdk-ts";
import { JsonRpcProvider, Connection as SuiConnection } from "@mysten/sui.js";
import { getCosmWasmClient } from "@sei-js/core";
import { Connection as SolanaConnection } from "@solana/web3.js";
import { Algodv2 } from "algosdk";
import { AptosClient } from "aptos";
import { ethers } from "ethers";
import { connect } from "near-api-js";
import { Provider as NearProvider } from "near-api-js/lib/providers";
import { NETWORKS } from "../../consts";
import { impossible } from "../../vaa";
import {
  Chain,
  Network,
  PlatformToChains,
} from "@wormhole-foundation/sdk-base";

export type ChainProvider<T extends Chain> = T extends "Algorand"
  ? Algodv2
  : T extends "Aptos"
  ? AptosClient
  : T extends PlatformToChains<"Evm">
  ? ethers.providers.JsonRpcProvider
  : T extends "Injective"
  ? ChainGrpcWasmApi
  : T extends "Near"
  ? Promise<NearProvider>
  : T extends "Sei"
  ? Promise<CosmWasmClient>
  : T extends "Solana"
  ? SolanaConnection
  : T extends "Sui"
  ? JsonRpcProvider
  : never;

export const getProviderForChain = <T extends Chain>(
  chain: T,
  network: Network,
  options?: { rpc?: string; [opt: string]: any }
): ChainProvider<T> => {
  const rpc = options?.rpc ?? NETWORKS[network][chain].rpc;
  if (!rpc) {
    throw new Error(`No ${network} rpc defined for ${chain}`);
  }

  switch (chain) {
    case "Solana":
      return new SolanaConnection(rpc, "confirmed") as ChainProvider<T>;
    case "Fogo":
      return new SolanaConnection(rpc, "confirmed") as ChainProvider<T>;
    case "Arbitrum":
    case "Avalanche":
    case "Base":
    case "Bsc":
    case "Celo":
    case "Ethereum":
    case "Fantom":
    case "Klaytn":
    case "Moonbeam":
    case "Optimism":
    case "Polygon":
    case "Scroll":
    case "Mantle":
    case "Xlayer":
    case "Linea":
    case "Berachain":
    case "Seievm":
    case "Sepolia":
    case "ArbitrumSepolia":
    case "BaseSepolia":
    case "OptimismSepolia":
    case "PolygonSepolia":
    case "Holesky":
      return new ethers.providers.JsonRpcProvider(rpc) as ChainProvider<T>;
    case "Injective": {
      const endpoints = getNetworkEndpoints(
        network === "Mainnet"
          ? InjectiveNetwork.MainnetK8s
          : InjectiveNetwork.TestnetK8s
      );
      return new ChainGrpcWasmApi(endpoints.grpc) as ChainProvider<T>;
    }
    case "Sei":
      return getCosmWasmClient(rpc) as ChainProvider<T>;
    case "Algorand": {
      const { token, port } = {
        ...{
          token:
            "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
          port: 4001,
        },
        ...options,
      };
      return new Algodv2(token, rpc, port) as ChainProvider<T>;
    }
    case "Near":
      return connect({
        networkId: NETWORKS[network].Near.networkId,
        nodeUrl: rpc,
        headers: {},
      }).then(({ connection }) => connection.provider) as ChainProvider<T>;
    case "Aptos":
      return new AptosClient(rpc) as ChainProvider<T>;
    case "Sui":
      return new JsonRpcProvider(
        new SuiConnection({ fullnode: rpc })
      ) as ChainProvider<T>;
    case "Btc":
    case "Osmosis":
    case "Pythnet":
    case "Wormchain":
    case "Cosmoshub":
    case "Evmos":
    case "Kujira":
    case "Neutron":
    case "Celestia":
    case "Stargaze":
    case "Seda":
    case "Dymension":
    case "Provenance":
    case "Unichain":
    case "HyperCore":
    case "Worldchain":
    case "Ink":
    case "HyperEVM":
    case "Monad":
    case "Mezo":
    case "Sonic":
    case "Converge":
    case "Plume":
    case "XRPLEVM":
    case "Plasma":
    case "CreditCoin":
    case "Noble":
      throw new Error(`${chain} not supported`);
    default:
      impossible(chain);
  }
};
