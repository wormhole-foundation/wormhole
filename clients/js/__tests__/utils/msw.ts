// This module provides logic to capture network calls by using 'msw' tool
import { setupServer } from "msw/node";
import { rest } from "msw";

let requests: any[] = [];

const logRequest = (req: any, res: any, ctx: any) => {
  console.log(`Received ${req.method} request at ${req.url.href}`);
  if (Object.keys(req.headers).length > 0) {
    console.log("Headers:", req.headers);
  }
  if (req.body) {
    console.log("Body:", req.body);
  }
  requests.push(req);
  return res(ctx.status(200), ctx.json({ message: "Success" }));
};

//NOTE: Capture all network traffic
export const handlers = [
  rest.get("*", logRequest),
  rest.post("*", logRequest),
  rest.put("*", logRequest),
  rest.delete("*", logRequest),
  rest.patch("*", logRequest),
];

export const server = setupServer(...handlers);
export { requests };
