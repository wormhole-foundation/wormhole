import { finalityThreshold } from "../src/constants/finality";

describe("Finality tests", function () {
  const mainnetFinality = finalityThreshold("Mainnet", "Ethereum");
  it("should correctly access values", function () {
    expect(mainnetFinality).toEqual(64);
  });
});

