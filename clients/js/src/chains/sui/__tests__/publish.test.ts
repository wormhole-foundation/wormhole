import { parseTestPublishDigest } from "../publish";

describe("parseTestPublishDigest", () => {
  it("extracts the digest from the `-q --json` payload", () => {
    const output = JSON.stringify({
      digest: "BwBvV26C79zyjUxwTCLVyZmjKsBqUGU7a9F5jySoPnVh",
      effects: { status: { status: "success" } },
    });

    expect(parseTestPublishDigest(output)).toBe(
      "BwBvV26C79zyjUxwTCLVyZmjKsBqUGU7a9F5jySoPnVh"
    );
  });

  it("throws when the output is not JSON", () => {
    expect(() => parseTestPublishDigest("error: command failed")).toThrow(
      /Unexpected non-JSON output/
    );
  });

  it("throws when the JSON has no digest field", () => {
    expect(() =>
      parseTestPublishDigest(JSON.stringify({ effects: {} }))
    ).toThrow(/No transaction digest/);
  });
});
