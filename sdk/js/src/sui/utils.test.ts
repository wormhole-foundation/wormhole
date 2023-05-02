import { unnormalizeSuiAddress } from "./utils";

describe("Sui utils tests", () => {
  test("Test unnormalizeSuiAddress", () => {
    const initial =
      "0x09bc8dd67bbbf59a43a9081d7166f9b41740c3a8ae868c4902d30eb247292ba4::coin::COIN";
    const expected =
      "0x9bc8dd67bbbf59a43a9081d7166f9b41740c3a8ae868c4902d30eb247292ba4::coin::COIN";
    expect(unnormalizeSuiAddress(initial)).toBe(expected);
  });
});
