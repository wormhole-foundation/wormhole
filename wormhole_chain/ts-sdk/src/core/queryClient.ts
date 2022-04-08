import {
  QueryClient,
  setupAuthExtension,
  setupBankExtension,
  setupGovExtension,
  setupIbcExtension,
  setupMintExtension,
  setupStakingExtension,
  setupTxExtension,
} from "@cosmjs/stargate";
import { Tendermint34Client } from "@cosmjs/tendermint-rpc";
import { Api as tokenbridgeApi } from "../modules/certusone.wormholechain.tokenbridge/rest";
import { Api as coreApi } from "../modules/certusone.wormholechain.wormhole/rest";

export type WormchainQueryClient = {
  coreClient: coreApi<any>;
  tokenBridgeClient: tokenbridgeApi<any>;
};

export function getWormholeQueryClient(
  lcdAddress: string,
  nodejs?: boolean
): WormchainQueryClient {
  if (nodejs) {
    var fetch = require("node-fetch");
    //@ts-ignore
    globalThis.fetch = fetch;
  }
  const coreClient = new coreApi({ baseUrl: lcdAddress });
  const tokenBridgeClient = new tokenbridgeApi({ baseUrl: lcdAddress });

  return { coreClient, tokenBridgeClient };
}

export async function getStargateQueryClient(tendermintAddress: string) {
  const tmClient = await Tendermint34Client.connect(tendermintAddress);
  const client = QueryClient.withExtensions(
    tmClient,
    setupTxExtension,
    setupGovExtension,
    setupIbcExtension,
    setupAuthExtension,
    setupBankExtension,
    setupMintExtension,
    setupStakingExtension
  );

  return client;
}
