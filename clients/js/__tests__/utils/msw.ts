// This module provides logic to capture network calls by using 'msw' tool
import { setupServer } from "msw/node";
import {
  AsyncResponseResolverReturnType,
  MockedRequest,
  MockedResponse,
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
) => AsyncResponseResolverReturnType<MockedResponse<Body>>;

const genericLogRequest: LogRequestFunction = async (req, res, ctx) => {
  console.log(`${req.method} request to ${req.url.toString()}`);
  if (Object.keys(req.headers).length > 0) {
    console.log("Headers:", req.headers.all());
  }
  if (req.body) {
    console.log("Body:", req.body);
  }
  const response = await ctx.fetch(req);
  const responseJSON = await response.json();
  console.log("Response:", responseJSON);

  requests.push(req);
  responses.push(responseJSON);

  //Return response back to execution thread
  return res(ctx.status(200), ctx.json(responseJSON));
};

//NOTE: Capture all network traffic
export const handlers = [
  rest.get("*", genericLogRequest),
  rest.post("*", genericLogRequest),
  rest.put("*", genericLogRequest),
  rest.delete("*", genericLogRequest),
  rest.patch("*", genericLogRequest),
];

export const server = setupServer(...handlers);
export { requests };
