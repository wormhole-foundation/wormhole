import { describe, expect, it } from "@jest/globals";
import { run_worm_command, run_worm_help_command } from "./utils-jest";
import { CHAINS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";

describe("worm chain-id", () => {
  describe("check arguments", () => {
    const FIRST_POSITIONAL_ARGUMENT = "<chain>";

    it(`should have correct positional arguments`, async () => {
      // Run the command module with --help as argument
      const output = run_worm_help_command("info chain-id");

      expect(output).toContain(FIRST_POSITIONAL_ARGUMENT);
    });
  });

  describe("check functionality", () => {
    type WormholeSDKChainName = keyof typeof CHAINS;

    const chains: WormholeSDKChainName[] = [
      "solana",
      "ethereum",
      "near",
      "wormchain",
      "aptos",
      "sui",
      "avalanche",
      "gnosis",
    ];

    chains.forEach((chain) => {
      it(`should return ${chain} chain-id correctly`, async () => {
        const output = run_worm_command(`info chain-id ${chain}`);
        expect(output).toContain(CHAINS[chain].toString());
      });
    });
  });
});
