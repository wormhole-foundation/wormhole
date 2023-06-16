import yargs from "yargs";
import { describe, expect, it, jest } from "@jest/globals";
import { run_worm_help_command } from "./utils-jest";

describe("worm chain-id", () => {
  describe("check arguments", () => {
    const FIRST_POSITIONAL_ARGUMENT = "<chain>";

    it(`should have correct positional arguments`, async () => {
      // Run the command module with --help as argument
      const output = run_worm_help_command("info chain-id");

      expect(output).toContain(FIRST_POSITIONAL_ARGUMENT);
    });
  });

  describe.skip("check functionality", () => {
    const chainIdCommand = require("../src/cmds/info/chainId");

    const SOLANA_CHAIN_ID = 1;
    const ETHEREUM_CHAIN_ID = 2;

    it(`should return solana chain-id correctly`, async () => {
      const consoleSpy = jest.spyOn(console, "log");

      const command = yargs.command(chainIdCommand).help();
      await command.parse(["chain-id", "solana"]);

      expect(consoleSpy).toBeCalledWith(SOLANA_CHAIN_ID);
    });

    it(`should return ethereum chain-id correctly`, async () => {
      const consoleSpy = jest.spyOn(console, "log");

      const command = yargs.command(chainIdCommand).help();
      await command.parse(["chain-id", "ethereum"]);

      expect(consoleSpy).toBeCalledWith(ETHEREUM_CHAIN_ID);
    });
  });
});
