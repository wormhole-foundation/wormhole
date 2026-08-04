import fs from "fs";
import os from "os";
import path from "path";
import { buildPackage, parseTestPublishDigest } from "../publish";

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

describe("buildPackage package path validation", () => {
  // Both cases are rejected before the `sui` binary is invoked, so these run
  // without a Sui toolchain or a network.
  it("throws when the package path does not exist", () => {
    const missing = path.join(os.tmpdir(), "worm-sui-does-not-exist-12345");
    expect(() => buildPackage(missing)).toThrow(/Package not found at/);
  });

  it("throws when the package path is a file rather than a directory", () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "worm-sui-"));
    const file = path.join(dir, "Move.toml");
    fs.writeFileSync(file, "[package]\n");

    try {
      expect(() => buildPackage(file)).toThrow(/is not a directory/);
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });
});
