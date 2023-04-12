import yargs from "yargs";
import { describe, expect, it } from "@jest/globals";

describe("worm chain-id", () => {
  describe("check arguments", () => {
    const FIRST_POSITIONAL_ARGUMENT = "<chain>";

    it(`should has ${FIRST_POSITIONAL_ARGUMENT} as first positional argument`, async () => {
      const command = await yargs.command(require("../cmds/chainId")).help();

      // Run the command module with --help as argument
      const output = await new Promise((resolve) => {
        command.parse("--help", (_err, _argv, output) => {
          resolve(output);
        });
      });

      expect(output).toContain(FIRST_POSITIONAL_ARGUMENT);
    });
  });
});
