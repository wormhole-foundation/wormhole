import yargs from "yargs";
import { describe, expect, it, jest } from "@jest/globals";

describe("worm contract", () => {
  describe("check arguments", () => {
    const FIRST_POSITIONAL_ARGUMENT = "<network>";
    const SECOND_POSITIONAL_ARGUMENT = "<chain>";
    const THIRD_POSITIONAL_ARGUMENT = "<module>";

    it(`should has correct positional arguments`, async () => {
      const command = await yargs
        .command(require("../cmds/contractAddress"))
        .help();

      // Run the command module with --help as argument
      const output = await new Promise((resolve) => {
        command.parse(["contract", "--help"], (_err, _argv, output) => {
          console.log("output", output);
          resolve(output);
        });
      });

      expect(output).toContain(FIRST_POSITIONAL_ARGUMENT);
      expect(output).toContain(SECOND_POSITIONAL_ARGUMENT);
      expect(output).toContain(THIRD_POSITIONAL_ARGUMENT);
    });
  });

  describe("check functionality", () => {
    const SOLANA_CORE_CONTRACT = "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth";
    const ETHEREUM_NFT_BRIDGE_CONTRACT =
      "0x6FFd7EdE62328b3Af38FCD61461Bbfc52F5651fE";

    it(`should return solana core mainnet contract correctly`, async () => {
      const consoleSpy = jest.spyOn(console, "log");

      const command = yargs.command(require("../cmds/contractAddress")).help();
      await command.parse(["contract", "mainnet", "solana", "Core"]);

      expect(consoleSpy).toBeCalledWith(SOLANA_CORE_CONTRACT);
    });

    it(`should return ethereum mainnet NFTBridge contract correctly`, async () => {
      const consoleSpy = jest.spyOn(console, "log");

      const command = yargs.command(require("../cmds/contractAddress")).help();
      await command.parse(["contract", "mainnet", "ethereum", "NFTBridge"]);

      expect(consoleSpy).toBeCalledWith(ETHEREUM_NFT_BRIDGE_CONTRACT);
    });
  });
});
