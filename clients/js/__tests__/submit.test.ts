import yargs from "yargs";
import { describe, expect, it } from "@jest/globals";

describe("worm submit", () => {
  describe("check arguments", () => {
    const FIRST_POSITIONAL_ARGUMENT = "<vaa>";
    const REQUIRED_FIRST_FLAG = "--network";
    const REQUIRED_SECOND_FLAG = "--chain";

    it(`should have correct positional arguments`, async () => {
      const command = await yargs.command(require("../src/cmds/submit")).help();

      // Run the command module with --help as argument
      const output = await new Promise((resolve) => {
        command.parse(["submit", "--help"], (_err, _argv, output) => {
          console.log("output", output);
          resolve(output);
        });
      });

      expect(output).toContain(FIRST_POSITIONAL_ARGUMENT);
      expect(output).toContain(REQUIRED_FIRST_FLAG);
      expect(output).toContain(REQUIRED_SECOND_FLAG);
    });
  });
});
