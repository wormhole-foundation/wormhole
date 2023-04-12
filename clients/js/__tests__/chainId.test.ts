import yargs from "yargs";
import { describe, expect, it, jest } from "@jest/globals";

describe("worm chain-id", () => {
  describe("check arguments", () => {
    const FIRST_POSITIONAL_ARGUMENT = "<chain>";

    it(`should have correct positional arguments`, async () => {
      const command = await yargs.command(require("../cmds/chainId")).help();

      // Run the command module with --help as argument
      const output = await new Promise((resolve) => {
        command.parse("--help", (_err, _argv, output) => {
          console.log("output", output);
          resolve(output);
        });
      });

      expect(output).toContain(FIRST_POSITIONAL_ARGUMENT);
    });
  });

  describe("check functionality", () => {
    const SOLANA_CHAIN_ID = 1;
    const ETHEREUM_CHAIN_ID = 2;

    it(`should return solana chain-id correctly`, async () => {
      const consoleSpy = jest.spyOn(console, "log");

      const command = yargs.command(require("../cmds/chainId")).help();
      await command.parse(["chain-id", "solana"]);

      expect(consoleSpy).toBeCalledWith(SOLANA_CHAIN_ID);
    });

    it(`should return ethereum chain-id correctly`, async () => {
      const consoleSpy = jest.spyOn(console, "log");

      const command = yargs.command(require("../cmds/chainId")).help();
      await command.parse(["chain-id", "ethereum"]);

      expect(consoleSpy).toBeCalledWith(ETHEREUM_CHAIN_ID);
    });
  });
});
