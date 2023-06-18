import yargs from "yargs";
import { describe, expect, it, jest } from "@jest/globals";
import { test_command_positional_args } from "./utils-cli";

describe("worm rpc", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["network", "chain"];

    test_command_positional_args("info rpc", args);
  });

  describe.skip("check functionality", () => {
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
