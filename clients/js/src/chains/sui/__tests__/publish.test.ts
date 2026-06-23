import { parseTestPublishDigest } from "../publish";

describe("parseTestPublishDigest", () => {
  it("extracts the digest from output with leading build lines", () => {
    const output = [
      "INCLUDING DEPENDENCY Sui",
      "BUILDING token_bridge",
      "Skipping dependency verification",
      JSON.stringify({
        digest: "BwBvV26C79zyjUxwTCLVyZmjKsBqUGU7a9F5jySoPnVh",
        effects: { status: { status: "success" } },
      }),
    ].join("\n");

    expect(parseTestPublishDigest(output)).toBe(
      "BwBvV26C79zyjUxwTCLVyZmjKsBqUGU7a9F5jySoPnVh"
    );
  });

  it("throws when there is no JSON object in the output", () => {
    expect(() => parseTestPublishDigest("error: command failed")).toThrow(
      /No JSON output/
    );
  });

  it("throws when the JSON has no digest field", () => {
    expect(() =>
      parseTestPublishDigest(JSON.stringify({ effects: {} }))
    ).toThrow(/No transaction digest/);
  });
});
