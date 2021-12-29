const { describe, expect, it } = require("@jest/globals");
const fs = require("fs");

describe("SDK installation", () => {
  it("does not import from file path", () => {
    const packageFile = fs.readFileSync("./package.json");
    const packageObj = JSON.parse(packageFile.toString());

    const sdkInstallation =
      packageObj?.dependencies?.["@certusone/wormhole-sdk"];
    expect(sdkInstallation && !sdkInstallation.includes("file")).toBe(true);
  });
});
