import {
  DEPRECATED_CHAINS,
  DeprecatedChain,
  deprecatedChainIdToName,
  deprecatedChainToId,
  deprecatedChainToPlatform,
  isDeprecatedChain,
  isDeprecatedChainId,
  isDeprecatedChainLike,
} from "./consts";

const NAMES = Object.keys(DEPRECATED_CHAINS) as DeprecatedChain[];

describe("DEPRECATED_CHAINS", () => {
  it("round-trips every id/name pair", () => {
    for (const name of NAMES) {
      const id = deprecatedChainToId(name);
      expect(deprecatedChainIdToName(id)).toBe(name);
    }
  });

  it("has no duplicate ids", () => {
    const ids = NAMES.map((name) => deprecatedChainToId(name));
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("only Terra and Xpla are Cosmwasm platform, the rest are Evm", () => {
    const cosmwasm = NAMES.filter(
      (name) => deprecatedChainToPlatform(name) === "Cosmwasm"
    );
    const evm = NAMES.filter(
      (name) => deprecatedChainToPlatform(name) === "Evm"
    );
    expect(cosmwasm.sort()).toEqual(["Terra", "Xpla"]);
    expect(evm.length).toBe(NAMES.length - 2);
  });
});

describe("isDeprecatedChain", () => {
  it("is true for every deprecated chain name", () => {
    for (const name of NAMES) {
      expect(isDeprecatedChain(name)).toBe(true);
    }
  });

  it("is false for a current chain, Terra2, and non-strings", () => {
    expect(isDeprecatedChain("Solana")).toBe(false);
    expect(isDeprecatedChain("Terra2")).toBe(false);
    expect(isDeprecatedChain(3)).toBe(false);
    expect(isDeprecatedChain(undefined)).toBe(false);
    expect(isDeprecatedChain("Nonsense")).toBe(false);
  });
});

describe("isDeprecatedChainId", () => {
  it("is true for every deprecated chain id", () => {
    for (const name of NAMES) {
      expect(isDeprecatedChainId(deprecatedChainToId(name))).toBe(true);
    }
  });

  it("is false for a current chain id, Terra2's id, and non-numbers", () => {
    expect(isDeprecatedChainId(1)).toBe(false); // Solana
    expect(isDeprecatedChainId(18)).toBe(false); // Terra2
    expect(isDeprecatedChainId("Terra")).toBe(false);
    expect(isDeprecatedChainId(undefined)).toBe(false);
  });
});

describe("isDeprecatedChainLike", () => {
  it("accepts both the name and the id form", () => {
    expect(isDeprecatedChainLike("Blast")).toBe(true);
    expect(isDeprecatedChainLike(36)).toBe(true);
    expect(isDeprecatedChainLike("Solana")).toBe(false);
    expect(isDeprecatedChainLike(1)).toBe(false);
  });
});
