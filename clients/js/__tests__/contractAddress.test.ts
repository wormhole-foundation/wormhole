import yargs from "yargs";
import { describe, expect, it, jest } from "@jest/globals";
import { run_worm_help_command } from "./utils-jest";

describe("worm info contract", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["network", "chain", "module"];

    it(`should have correct positional arguments`, async () => {
      // Run the command module with --help as argument
      const output = run_worm_help_command("info contract");
      const expectedPositionalArgs = args.map((arg) => `<${arg}>`).join(" ");

      expect(output).toContain(`info contract ${expectedPositionalArgs}`);
    });
  });

  describe.skip("check functionality", () => {
    const SOLANA_CORE_CONTRACT = "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth";
    const ETHEREUM_NFT_BRIDGE_CONTRACT =
      "0x6FFd7EdE62328b3Af38FCD61461Bbfc52F5651fE";

    it(`should return solana core mainnet contract correctly`, async () => {
      const consoleSpy = jest.spyOn(console, "log");

      const command = yargs
        .command(require("../src/cmds/contractAddress"))
        .help();
      await command.parse(["contract", "mainnet", "solana", "Core"]);

      expect(consoleSpy).toBeCalledWith(SOLANA_CORE_CONTRACT);
    });

    it(`should return ethereum mainnet NFTBridge contract correctly`, async () => {
      const consoleSpy = jest.spyOn(console, "log");

      const command = yargs
        .command(require("../src/cmds/contractAddress"))
        .help();
      await command.parse(["contract", "mainnet", "ethereum", "NFTBridge"]);

      expect(consoleSpy).toBeCalledWith(ETHEREUM_NFT_BRIDGE_CONTRACT);
    });
  });
});
