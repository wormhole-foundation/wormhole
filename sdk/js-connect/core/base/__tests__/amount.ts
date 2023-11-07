import { normalizeAmount } from "../src/";

describe("Amount Tests", function () {
  const cases: [number | string, bigint, bigint][] = [
    [1, 18n, BigInt(1 + "0".repeat(18))],
    [0, 18n, BigInt(0)],
    [1, 2n, BigInt(1 + "0".repeat(2))],
    [3.2, 2n, BigInt(320)],
    ["1.4", 12n, BigInt(1400000000000)],
    ["0.0001", 12n, BigInt(100000000)],
    ["0", 2n, BigInt(0)],
    // should we throw on negative?
    [-3, 2n, BigInt(-300)],
    ["-3", 2n, BigInt(-300)],
  ];

  const badCases: [number | string, bigint][] = [
    ["0.000001", 2n],
    ["-0.000001", 2n],
    ["3", -2n],
  ];

  it("should correctly normalize values", function () {
    for (const [amt, dec, expected] of cases) {
      const actual = normalizeAmount(amt, dec);
      expect(actual).toEqual(expected);
    }
  });

  it("should correctly fail on unexpected values", function () {
    for (const [amt, dec] of badCases) {
      const actual = () => normalizeAmount(amt, dec);
      expect(actual).toThrow();
    }
  });
});
