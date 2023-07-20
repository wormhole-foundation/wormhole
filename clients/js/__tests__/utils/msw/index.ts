// This module provides logic to capture network calls by using 'msw' tool
import { setupServer } from "msw/node";
import { rest } from "msw";
import { Request, Response } from "./types";
import {
  ethereumRequestHandler,
  genericRequestHandler,
  solanaRequestHandler,
} from "./handlers";

let requests: Request[] = [];
let responses: Response[] = [];

//NOTE: Capture all network traffic
const handlers = [
  // Interceptors
  rest.post("https://api.devnet.solana.com/", solanaRequestHandler),
  rest.post("https://rpc.ankr.com/eth", ethereumRequestHandler),

  // Loggers
  rest.get("*", genericRequestHandler),
  rest.post("*", genericRequestHandler),
  rest.put("*", genericRequestHandler),
  rest.delete("*", genericRequestHandler),
  rest.patch("*", genericRequestHandler),
];

const server = setupServer(...handlers);

export { server, requests, responses };
