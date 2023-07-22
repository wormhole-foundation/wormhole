// This module provides logic to capture network calls by using 'msw' tool
import { setupServer } from "msw/node";
import { rest } from "msw";
import { Request, Response } from "./types";
import {
  algorandRequestHandler,
  aptosRequestHandler,
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
  "acala",
  "arbitrum",
  "aurora",
  "avalanche",
  "bsc",
  "celo",
  "fantom",
  "gnosis",
  "karura",
  "klaytn",
  "moonbeam",
  "oasis",
  "optimism",
  "polygon",
].map((chain) => {
  // @ts-ignore
  const rpc = NETWORKS["MAINNET"][chain].rpc;
  return rest.post(rpc, evmRequestHandler);
});

//NOTE: Capture all network traffic
const handlers = [
  // Interceptors
  ...evmHandlers,
  rest.post(NETWORKS["TESTNET"]["solana"].rpc, solanaRequestHandler),
  rest.post(`${NETWORKS["MAINNET"]["sui"].rpc}`, suiRequestHandler),
  rest.post(`${NETWORKS["MAINNET"]["near"].rpc}`, nearRequestHandler),
  rest.post(
    `${NETWORKS["MAINNET"]["algorand"].rpc}/v2/transactions`,
    algorandRequestHandler
  ),
  rest.post(
    `${NETWORKS["MAINNET"]["aptos"].rpc}/transactions/simulate`,
    aptosRequestHandler
  ),
  rest.post(
    `${NETWORKS["MAINNET"]["aptos"].rpc}/transactions`,
    aptosRequestHandler
  ),

  // Loggers
  rest.get("*", genericRequestHandler),
  rest.post("*", genericRequestHandler),
  rest.put("*", genericRequestHandler),
  rest.patch("*", genericRequestHandler),
];

const server = setupServer(...handlers);

export { server, requests, responses };
