// This module provides logic to capture network calls by using 'msw' tool
import { setupServer } from "msw/node";
import {
  AsyncResponseResolverReturnType,
  MockedRequest,
  ResponseComposition,
  RestContext,
  rest,
} from "msw";

let requests: MockedRequest<Body>[] = [];
let responses: ResponseComposition<Body>[] = [];

type Body = Record<string, any>;
type LogRequestFunction = (
  req: MockedRequest<Body>,
  res: ResponseComposition<Body>,
  context: RestContext
) => AsyncResponseResolverReturnType<any>;

const logRequest = (req: MockedRequest<Body>) => {
  console.log(`${req.method} request to ${req.url.toString()}`);
  if (Object.keys(req.headers).length > 0) {
    console.log("Headers:", req.headers.all());
  }
  if (req.body) {
    console.log("Body:", req.body);
  }
  requests.push(req);
};

const genericRequestHandler: LogRequestFunction = async (req, res, ctx) => {
  logRequest(req);

  const response = await ctx.fetch(req);
  const responseJSON = await response.json();
  console.log("Response:", responseJSON);
  responses.push(responseJSON);

  //Return response back to execution thread
  return res(ctx.status(200), ctx.json(responseJSON));
};

const solanaRequestHandler: LogRequestFunction = async (req, res, ctx) => {
  logRequest(req);

  // Avoid sending transaction to network, send error instead (to force stop execution)
  if (req.body && req.body.method === "sendTransaction") {
    return res(
      ctx.status(200),
      // mock response with error
      ctx.json({
        jsonrpc: "2.0",
        error: {
          code: -32002,
          message: "Transaction signature verification failed",
        },
        id: "9b282a84-c613-4a34-b5b8-5c6a3fd5f352",
      })
    );
  } else {
    return await genericRequestHandler(req, res, ctx);
  }
};

const ethereumRequestHandler: LogRequestFunction = async (req, res, ctx) => {
  logRequest(req);

  let response;
  const method = req.body.method;

  switch (method) {
    case "eth_estimateGas":
      response = {
        jsonrpc: "2.0",
        id: 1,
        result: "0x5208", // hexadecimal representation of gas estimation
      };
      break;
    case "eth_sendRawTransaction":
      response = {
        jsonrpc: "2.0",
        id: 1,
        result:
          "0x9fc76417374aa880d4449a1f7f31ec597f00b1f6f3dd2d66f4c9c6c445836d8b",
      };
      break;
    default:
      break;
  }

  if (!response) {
    return await genericRequestHandler(req, res, ctx);
  }

  return res(ctx.status(200), ctx.json(response));
};

//NOTE: Capture all network traffic
export const handlers = [
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

export const server = setupServer(...handlers);
export { requests };
