// This module provides logic to capture network calls by using 'msw' tool
import { setupServer } from "msw/node";
import { rest } from "msw";
import { Request, Response } from "./types";
import {
  algorandRequestHandler,
  aptosRequestHandler,
  cosmwasmRequestHandler,
  evmRequestHandler,
  genericRequestHandler,
  nearRequestHandler,
  solanaRequestHandler,
  suiRequestHandler,
} from "./handlers";
import { NETWORKS } from "../../../src/consts";

let requests: Request[] = [];
let responses: Response[] = [];

const evmHandlers = [
  "ethereum",
  "arbitrum",
  "aurora",
  "avalanche",
  "bsc",
  "celo",
  "fantom",
  "gnosis",
  "klaytn",
  "moonbeam",
  "oasis",
  "optimism",
  "polygon",
  "karura",
  "acala",
  "sepolia",
  "neon",
].map((chain) => {
  const testnetEvmChains = ["sepolia", "neon"];
  const network = testnetEvmChains.includes(chain) ? "TESTNET" : "MAINNET";
  // @ts-ignore
  const rpc = NETWORKS[network][chain].rpc;
  return rest.post(rpc, evmRequestHandler);
});

const cosmwasmHandlers = ["xpla", "sei", "injective", "terra2", "terra"]
  .map((chain) => {
    const network = chain === "sei" ? "TESTNET" : "MAINNET";
    // @ts-ignore
    const rpc = `${NETWORKS[network][chain].rpc}/*`;

    return [
      rest.get(rpc, cosmwasmRequestHandler),
      rest.post(rpc, cosmwasmRequestHandler),
    ];
  })
  .flat();

//NOTE: Capture all network traffic
const handlers = [
  // Interceptors
  ...evmHandlers,
  rest.post(`${NETWORKS["TESTNET"]["solana"].rpc}`, solanaRequestHandler),
  rest.post(`${NETWORKS["MAINNET"]["sui"].rpc}`, suiRequestHandler),
  rest.post(`${NETWORKS["MAINNET"]["near"].rpc}`, nearRequestHandler),
  rest.post(`${NETWORKS["MAINNET"]["algorand"].rpc}/*`, algorandRequestHandler),
  rest.post(`${NETWORKS["MAINNET"]["aptos"].rpc}/*`, aptosRequestHandler),
  ...cosmwasmHandlers,
  rest.get(
    "https://k8s.mainnet.lcd.injective.network/*",
    cosmwasmRequestHandler
  ),
  rest.post(
    "https://k8s.mainnet.lcd.injective.network/*",
    cosmwasmRequestHandler
  ),

  // Loggers
  rest.get("*", genericRequestHandler),
  rest.post("*", genericRequestHandler),
  rest.put("*", genericRequestHandler),
  rest.patch("*", genericRequestHandler),
];

const server = setupServer(...handlers);

export { server, requests, responses };