import { CosmWasmClient } from "@cosmjs/cosmwasm-stargate";
import {
  Network as InjectiveNetwork,
  getNetworkEndpoints,
} from "@injectivelabs/networks";
import { ChainGrpcWasmApi } from "@injectivelabs/sdk-ts";
import { JsonRpcProvider, Connection as SuiConnection } from "@mysten/sui.js";
import { getCosmWasmClient } from "@sei-js/core";
import { Connection as SolanaConnection } from "@solana/web3.js";
import { LCDClient as TerraLCDClient } from "@terra-money/terra.js";
import { LCDClient as XplaLCDClient } from "@xpla/xpla.js";
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
  : T extends "Terra" | "Terra2"
  ? TerraLCDClient
  : T extends "Sei"
  ? Promise<CosmWasmClient>
  : T extends "Solana"
  ? SolanaConnection
  : T extends "Sui"
  ? JsonRpcProvider
  : T extends "Xpla"
  ? XplaLCDClient
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
    case "Acala":
    case "Arbitrum":
    case "Aurora":
    case "Avalanche":
    case "Base":
    case "Bsc":
    case "Celo":
    case "Ethereum":
    case "Fantom":
    case "Gnosis":
    case "Karura":
    case "Klaytn":
    case "Moonbeam":
    case "Neon":
    case "Oasis":
    case "Optimism":
    case "Polygon":
    // case "Rootstock":
    case "Scroll":
    case "Mantle":
    case "Blast":
    case "Xlayer":
    case "Linea":
    case "Berachain":
    case "Snaxchain":
    case "Seievm":
    case "Sepolia":
    case "ArbitrumSepolia":
    case "BaseSepolia":
    case "OptimismSepolia":
    case "PolygonSepolia":
    case "Holesky":
      return new ethers.providers.JsonRpcProvider(rpc) as ChainProvider<T>;
    case "Terra":
    case "Terra2":
      const chain_id =
        chain === "Terra"
          ? NETWORKS[network].Terra.chain_id
          : NETWORKS[network].Terra2.chain_id;
      return new TerraLCDClient({
        URL: rpc,
        chainID: chain_id,
        isClassic: chain === "Terra",
      }) as ChainProvider<T>;
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
    case "Xpla": {
      const chainId = NETWORKS[network].Xpla.chain_id;
      if (!chainId) {
        throw new Error(`No ${network} chain ID defined for XPLA.`);
      }

      return new XplaLCDClient({
        URL: rpc,
        chainID: chainId,
      }) as ChainProvider<T>;
    }
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
    case "Rootstock":
      throw new Error(`${chain} not supported`);
    default:
      impossible(chain);
  }
};
