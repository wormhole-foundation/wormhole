import { CosmWasmClient } from "@cosmjs/cosmwasm-stargate";
import {
  Network as InjectiveNetwork,
  getNetworkEndpoints,
} from "@injectivelabs/networks";
import { ChainGrpcWasmApi } from "@injectivelabs/sdk-ts";
import { SuiGrpcClient } from "@mysten/sui/grpc";
import { getSuiNetwork } from "../sui/utils";
import { getCosmWasmClient } from "@sei-js/core";
import { Connection as SolanaConnection } from "@solana/web3.js";
import type { LCDClient as TerraLCDClient } from "@terra-money/terra.js";
import { Algodv2 } from "algosdk";
import { AptosClient } from "aptos";
import { ethers } from "ethers";
import { connect } from "near-api-js";
import { Provider as NearProvider } from "near-api-js/lib/providers";
import { NETWORKS } from "../../consts";
import { getTerra2Client } from "../terra2";
import { Network, PlatformToChains } from "@wormhole-foundation/sdk-base";
import { CliChain, cliChainToPlatform, getChainRpc } from "../../utils";

export type ChainProvider<T extends CliChain> = T extends "Algorand"
  ? Algodv2
  : T extends "Aptos"
  ? AptosClient
  : T extends PlatformToChains<"Evm">
  ? ethers.providers.JsonRpcProvider
  : T extends "Injective"
  ? ChainGrpcWasmApi
  : T extends "Near"
  ? Promise<NearProvider>
  : T extends "Terra2"
  ? TerraLCDClient
  : T extends "Sei"
  ? Promise<CosmWasmClient>
  : T extends "Solana"
  ? SolanaConnection
  : T extends "Sui"
  ? SuiGrpcClient
  : never;

export const getProviderForChain = <T extends CliChain>(
  chain: T,
  network: Network,
  options?: { rpc?: string; [opt: string]: any }
): ChainProvider<T> => {
  const rpc = options?.rpc ?? getChainRpc(network, chain);
  if (!rpc) {
    throw new Error(`No ${network} rpc defined for ${chain}`);
  }

  if (cliChainToPlatform(chain) === "Evm") {
    return new ethers.providers.JsonRpcProvider(rpc) as ChainProvider<T>;
  }

  switch (chain) {
    case "Solana":
      return new SolanaConnection(rpc, "confirmed") as ChainProvider<T>;
    case "Terra2":
      return getTerra2Client(network, rpc) as ChainProvider<T>;
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
      return new SuiGrpcClient({
        network: getSuiNetwork(network),
        baseUrl: rpc,
      }) as ChainProvider<T>;
    default:
      throw new Error(`${chain} not supported`);
  }
};
