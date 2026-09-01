import { BridgeImplementation__factory } from "@certusone/wormhole-sdk/lib/esm/ethers-contracts";
import {
  DEPRECATED_CHAINS,
  DeprecatedChain,
  deprecatedChainToId,
} from "./chains/deprecated";
import { queryRegistrationsEvm } from "./evm";

const DEPRECATED_NAMES = Object.keys(DEPRECATED_CHAINS) as DeprecatedChain[];
const ZERO_REGISTRATION = "0x" + "00".repeat(32);
// a registration left behind on-chain for Blast (chain id 36), which the SDK
// has since dropped
const BLAST_REGISTRATION = "0x" + "ab".repeat(32);

describe("queryRegistrationsEvm", () => {
  const queriedChainIds: number[] = [];

  beforeEach(() => {
    queriedChainIds.length = 0;
    jest.spyOn(BridgeImplementation__factory, "connect").mockReturnValue({
      bridgeContracts: async (chainId: number) => {
        queriedChainIds.push(chainId);
        return chainId === deprecatedChainToId("Blast")
          ? BLAST_REGISTRATION
          : ZERO_REGISTRATION;
      },
    } as any);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("queries the ids of chains the SDK dropped", async () => {
    await queryRegistrationsEvm("Mainnet", "Ethereum", "TokenBridge");
    for (const name of DEPRECATED_NAMES) {
      expect(queriedChainIds).toContain(deprecatedChainToId(name));
    }
  });

  it("surfaces a lingering on-chain registration under the deprecated chain's name", async () => {
    const results: any = await queryRegistrationsEvm(
      "Mainnet",
      "Ethereum",
      "TokenBridge"
    );
    expect(results.Blast).toBe(BLAST_REGISTRATION);
    expect(results.Terra).toBe(ZERO_REGISTRATION);
    // the queried chain itself is never in the results
    expect(results.Ethereum).toBeUndefined();
  });
});
