import {
  CHAIN_ID_ALGORAND,
  CHAIN_ID_APTOS,
  CHAIN_ID_INJECTIVE,
  CHAIN_ID_NEAR,
  CHAIN_ID_SEI,
  CHAIN_ID_SOLANA,
  CHAIN_ID_SUI,
  CHAIN_ID_TERRA,
  CHAIN_ID_TERRA2,
  CHAIN_ID_XPLA,
  ChainId,
  ChainName,
  EVMChainId,
  EVMChainName,
  coalesceChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
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
import { Network } from "../../utils";
import { impossible } from "../../vaa";

export type ChainProvider<T extends ChainId | ChainName> = T extends
  | "algorand"
  | typeof CHAIN_ID_ALGORAND
  ? Algodv2
  : T extends "aptos" | typeof CHAIN_ID_APTOS
  ? AptosClient
  : T extends EVMChainName | EVMChainId
  ? ethers.providers.JsonRpcProvider
  : T extends "injective" | typeof CHAIN_ID_INJECTIVE
  ? ChainGrpcWasmApi
  : T extends "near" | typeof CHAIN_ID_NEAR
  ? Promise<NearProvider>
  : T extends
      | "terra"
      | "terra2"
      | typeof CHAIN_ID_TERRA
      | typeof CHAIN_ID_TERRA2
  ? TerraLCDClient
  : T extends "sei" | typeof CHAIN_ID_SEI
  ? Promise<CosmWasmClient>
  : T extends "solana" | typeof CHAIN_ID_SOLANA
  ? SolanaConnection
  : T extends "sui" | typeof CHAIN_ID_SUI
  ? JsonRpcProvider
  : T extends "xpla" | typeof CHAIN_ID_XPLA
  ? XplaLCDClient
  : never;

export const getProviderForChain = <T extends ChainId | ChainName>(
  chain: T,
  network: Network,
  options?: { rpc?: string; [opt: string]: any }
): ChainProvider<T> => {
  const chainName = coalesceChainName(chain);
  const rpc = options?.rpc ?? NETWORKS[network][chainName].rpc;
  if (!rpc) {
    throw new Error(`No ${network} rpc defined for ${chainName}`);
  }

  switch (chainName) {
    case "unset":
      throw new Error("Chain not set");
    case "solana":
      return new SolanaConnection(rpc, "confirmed") as ChainProvider<T>;
    case "acala":
    case "arbitrum":
    case "aurora":
    case "avalanche":
    case "base":
    case "bsc":
    case "celo":
    case "ethereum":
    case "fantom":
    case "gnosis":
    case "karura":
    case "klaytn":
    case "moonbeam":
    case "neon":
    case "oasis":
    case "optimism":
    case "polygon":
    // case "rootstock":
    case "scroll":
    case "mantle":
    case "blast":
    case "xlayer":
    case "linea":
    case "berachain":
    case "seievm":
    case "sepolia":
    case "arbitrum_sepolia":
    case "base_sepolia":
    case "optimism_sepolia":
    case "polygon_sepolia":
    case "holesky":
      return new ethers.providers.JsonRpcProvider(rpc) as ChainProvider<T>;
    case "terra":
    case "terra2":
      return new TerraLCDClient({
        URL: rpc,
        chainID: NETWORKS[network][chainName].chain_id,
        isClassic: chainName === "terra",
      }) as ChainProvider<T>;
    case "injective": {
      const endpoints = getNetworkEndpoints(
        network === "MAINNET"
          ? InjectiveNetwork.MainnetK8s
          : InjectiveNetwork.TestnetK8s
      );
      return new ChainGrpcWasmApi(endpoints.grpc) as ChainProvider<T>;
    }
    case "sei":
      return getCosmWasmClient(rpc) as ChainProvider<T>;
    case "xpla": {
      const chainId = NETWORKS[network].xpla.chain_id;
      if (!chainId) {
        throw new Error(`No ${network} chain ID defined for XPLA.`);
      }

      return new XplaLCDClient({
        URL: rpc,
        chainID: chainId,
      }) as ChainProvider<T>;
    }
    case "algorand": {
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
    case "near":
      return connect({
        networkId: NETWORKS[network].near.networkId,
        nodeUrl: rpc,
        headers: {},
      }).then(({ connection }) => connection.provider) as ChainProvider<T>;
    case "aptos":
      return new AptosClient(rpc) as ChainProvider<T>;
    case "sui":
      return new JsonRpcProvider(
        new SuiConnection({ fullnode: rpc })
      ) as ChainProvider<T>;
    case "btc":
    case "osmosis":
    case "pythnet":
    case "wormchain":
    case "cosmoshub":
    case "evmos":
    case "kujira":
    case "neutron":
    case "celestia":
    case "stargaze":
    case "seda":
    case "dymension":
    case "provenance":
    case "rootstock":
      throw new Error(`${chainName} not supported`);
    default:
      impossible(chainName);
  }
};
