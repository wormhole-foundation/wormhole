import { requests, responses } from "./index";
import { LogRequestFunction, Request } from "./types";

export const logRequest = (req: Request) => {
  console.log(`${req.method} request to ${req.url.toString()}`);
  if (Object.keys(req.headers).length > 0) {
    console.log("Headers:", req.headers.all());
  }
  if (req.body) {
    console.log("Body:", req.body);
  }
  requests.push(req);
};

export const genericRequestHandler: LogRequestFunction = async (
  req,
  res,
  ctx
) => {
  logRequest(req);

  const response = await ctx.fetch(req);
  const responseJSON = await response.json();
  console.log("Response:", responseJSON);
  responses.push(responseJSON);

  //Return response back to execution thread
  return res(ctx.status(200), ctx.json(responseJSON));
};

export const solanaRequestHandler: LogRequestFunction = async (
  req,
  res,
  ctx
) => {
  logRequest(req);

  if (req.body.method === "sendTransaction") {
    // Avoid sending transaction to network, send error instead (to force stop execution)
    return res(
      ctx.status(200),
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

export const aptosRequestHandler: LogRequestFunction = async (
  req,
  res,
  ctx
) => {
  logRequest(req);

  if (req.url.toString().includes("/transactions/simulate")) {
    // Sending an empty simulated transaction, to avoid a 'Failed to deserialize input into SignedTransaction' runtime error on validation
    return res(ctx.status(200), ctx.json([]));
  } else {
    return await genericRequestHandler(req, res, ctx);
  }
};

export const evmRequestHandler: LogRequestFunction = async (req, res, ctx) => {
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
