import { SuiGrpcClient } from "@mysten/sui/grpc";
import { SuiGraphQLClient } from "@mysten/sui/graphql";
import { fromBase64, toBase64 } from "@mysten/sui/utils";

describe("v2 SDK toolchain smoke", () => {
  it("loads the gRPC client class", () => {
    expect(typeof SuiGrpcClient).toBe("function");
  });

  it("loads the GraphQL client class", () => {
    expect(typeof SuiGraphQLClient).toBe("function");
  });

  it("round-trips base64 via @mysten/sui/utils", () => {
    const bytes = new Uint8Array([1, 2, 3, 4]);
    expect(fromBase64(toBase64(bytes))).toEqual(bytes);
  });
});
