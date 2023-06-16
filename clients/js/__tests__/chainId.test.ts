import { describe, expect, it } from "@jest/globals";
import { run_worm_command, run_worm_help_command } from "./utils-jest";

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
    const SOLANA_CHAIN_ID = "1";
    const ETHEREUM_CHAIN_ID = "2";

    it(`should return solana chain-id correctly`, async () => {
      const output = run_worm_command("info chain-id solana");
      expect(output).toContain(SOLANA_CHAIN_ID);
    });

    it(`should return ethereum chain-id correctly`, async () => {
      const output = run_worm_command("info chain-id ethereum");
      expect(output).toContain(ETHEREUM_CHAIN_ID);
    });
  });
});
