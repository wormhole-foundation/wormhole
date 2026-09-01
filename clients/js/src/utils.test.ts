import { chains } from "@wormhole-foundation/sdk-base";
import {
  DEPRECATED_CHAINS,
  DeprecatedChain,
  deprecatedChainToPlatform,
} from "./chains/deprecated";
import { TERRA2 } from "./chains/terra2";
import {
  CLI_CHAINS,
  assertLiveChain,
  chainToCliChain,
  cliChainIdToChain,
  cliChainToChainId,
  cliChainToPlatform,
  getChainRpc,
  getCoreContract,
  getNftBridgeContract,
  getRelayerContract,
  getTokenBridgeContract,
  toCliChain,
} from "./utils";

const DEPRECATED_NAMES = Object.keys(DEPRECATED_CHAINS) as DeprecatedChain[];

describe("CLI_CHAINS", () => {
  it("is the SDK's chains, plus Terra2, plus every deprecated chain", () => {
    expect(CLI_CHAINS.length).toBe(chains.length + 1 + DEPRECATED_NAMES.length);
    expect(new Set(CLI_CHAINS).size).toBe(CLI_CHAINS.length);
    for (const name of DEPRECATED_NAMES) {
      expect(CLI_CHAINS).toContain(name);
    }
    expect(CLI_CHAINS).toContain(TERRA2);
  });
});

describe("chainToCliChain", () => {
  it("resolves deprecated chains case-insensitively", () => {
    expect(chainToCliChain("blast")).toBe("Blast");
    expect(chainToCliChain("BLAST")).toBe("Blast");
    expect(chainToCliChain("Xpla")).toBe("Xpla");
  });

  it("still resolves Terra2 and a multi-capital current chain", () => {
    expect(chainToCliChain("terra2")).toBe("Terra2");
    expect(chainToCliChain("hyperevm")).toBe("HyperEVM");
  });

  it("throws on an unrecognized chain", () => {
    expect(() => chainToCliChain("not-a-chain")).toThrow("Invalid chain");
  });
});

describe("cliChainIdToChain / cliChainToChainId", () => {
  it("round-trips every chain the CLI supports", () => {
    for (const chain of CLI_CHAINS) {
      const id = cliChainToChainId(chain);
      expect(cliChainIdToChain(id)).toBe(chain);
    }
  });

  it("resolves a deprecated chain id to its name", () => {
    expect(cliChainIdToChain(36)).toBe("Blast"); // dropped from the SDK
    expect(cliChainIdToChain(3)).toBe("Terra"); // Terra Classic
  });

  it("throws on an id nothing (SDK, Terra2, or deprecated) recognizes", () => {
    expect(() => cliChainIdToChain(999999)).toThrow(
      "Unknown Wormhole chain id: 999999"
    );
  });
});

describe("toCliChain", () => {
  it("normalizes deprecated chains from either an id or a name", () => {
    expect(toCliChain(36)).toBe("Blast");
    expect(toCliChain("Blast")).toBe("Blast");
  });

  it("still normalizes Terra2 and current chains", () => {
    expect(toCliChain(18)).toBe("Terra2");
    expect(toCliChain("Solana")).toBe("Solana");
  });
});

describe("cliChainToPlatform", () => {
  it("matches each deprecated chain's declared platform", () => {
    for (const name of DEPRECATED_NAMES) {
      expect(cliChainToPlatform(name)).toBe(deprecatedChainToPlatform(name));
    }
  });

  it("still classifies Terra2 and the Solana-platform chains", () => {
    expect(cliChainToPlatform(TERRA2)).toBe("Cosmwasm");
    expect(cliChainToPlatform("Solana")).toBe("Solana");
    expect(cliChainToPlatform("Pythnet")).toBe("Solana");
    expect(cliChainToPlatform("Fogo")).toBe("Solana");
  });
});

describe("assertLiveChain", () => {
  it("throws for every deprecated chain, echoing the given reason", () => {
    for (const name of DEPRECATED_NAMES) {
      expect(() => assertLiveChain(name, "not supported")).toThrow(
        `${name} was dropped from the SDK and has no live bridge; not supported`
      );
    }
  });

  it("accepts current chains and Terra2", () => {
    expect(() => assertLiveChain("Solana", "not supported")).not.toThrow();
    expect(() => assertLiveChain(TERRA2, "not supported")).not.toThrow();
  });
});

describe("per-chain config getters", () => {
  it("report every deprecated chain as unconfigured", () => {
    for (const name of DEPRECATED_NAMES) {
      expect(getChainRpc("Mainnet", name)).toBeUndefined();
      expect(getCoreContract("Mainnet", name)).toBeUndefined();
      expect(getTokenBridgeContract("Mainnet", name)).toBeUndefined();
      expect(getNftBridgeContract("Mainnet", name)).toBeUndefined();
      expect(getRelayerContract("Mainnet", name)).toBeUndefined();
    }
  });

  it("still resolves Terra2 and a current chain", () => {
    expect(getChainRpc("Mainnet", TERRA2)).toBeDefined();
    expect(getChainRpc("Mainnet", "Solana")).toBeDefined();
    expect(getCoreContract("Mainnet", "Solana")).toBeDefined();
  });
});
