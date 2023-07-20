import {
  AsyncResponseResolverReturnType,
  MockedRequest,
  ResponseComposition,
  RestContext,
} from "msw";

export type Body = Record<string, any>;

export type Request = MockedRequest<Body>;

export type Response = ResponseComposition<Body>;

export type LogRequestFunction = (
  req: Request,
  res: Response,
  context: RestContext
) => AsyncResponseResolverReturnType<any>;
