import yargs from "yargs";
import { describe, expect, it, jest } from "@jest/globals";

describe("worm rpc", () => {
  describe("check arguments", () => {
    const FIRST_POSITIONAL_ARGUMENT = "<network>";
    const SECOND_POSITIONAL_ARGUMENT = "<chain>";

    it(`should have correct positional arguments`, async () => {
      const command = await yargs.command(require("../src/cmds/rpc")).help();

      // Run the command module with --help as argument
      const output = await new Promise((resolve) => {
        command.parse(["rpc", "--help"], (_err, _argv, output) => {
          console.log("output", output);
          resolve(output);
        });
      });

      expect(output).toContain(FIRST_POSITIONAL_ARGUMENT);
      expect(output).toContain(SECOND_POSITIONAL_ARGUMENT);
    });
  });

  describe("check functionality", () => {
    const SOLANA_RPC_URL = "https://api.mainnet-beta.solana.com";
    const ETHEREUM_RPC_URL = "https://rpc.ankr.com/eth";

    it(`should return solana mainnet rpc correctly`, async () => {
      const consoleSpy = jest.spyOn(console, "log");

      const command = yargs.command(require("../src/cmds/rpc")).help();
      await command.parse(["rpc", "mainnet", "solana"]);

      expect(consoleSpy).toBeCalledWith(SOLANA_RPC_URL);
    });

    it(`should return ethereum mainnet rpc correctly`, async () => {
      const consoleSpy = jest.spyOn(console, "log");

      const command = yargs.command(require("../src/cmds/rpc")).help();
      await command.parse(["rpc", "mainnet", "ethereum"]);

      expect(consoleSpy).toBeCalledWith(ETHEREUM_RPC_URL);
    });
  });
});
