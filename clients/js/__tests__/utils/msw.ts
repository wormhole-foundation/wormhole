// This module provides logic to capture network calls by using 'msw' tool
import { setupServer } from "msw/node";
import {
  AsyncResponseResolverReturnType,
  MockedRequest,
  ResponseComposition,
  RestContext,
  rest,
} from "msw";

let requests: any[] = [];
let responses: any[] = [];

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

//NOTE: Capture all network traffic
export const handlers = [
  // Interceptors
  rest.post("https://api.devnet.solana.com/", solanaRequestHandler),

  // Loggers
  rest.get("*", genericRequestHandler),
  rest.post("*", genericRequestHandler),
  rest.put("*", genericRequestHandler),
  rest.delete("*", genericRequestHandler),
  rest.patch("*", genericRequestHandler),
];

export const server = setupServer(...handlers);
export { requests };
