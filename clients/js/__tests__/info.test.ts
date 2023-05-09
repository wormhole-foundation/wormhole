import yargs from "yargs";
import { describe, expect, it } from "@jest/globals";

describe("worm info", () => {
  describe("check commands", () => {
    const FIRST_COMMAND = "info chain-id";
    const SECOND_COMMAND = "info rpc";
    const THIRD_COMMAND = "info contract";

    it(`should have correct commands in namespace`, async () => {
      const command = await yargs.command(require("../src/cmds/info")).help();

      // Run the command module with --help as argument
      const output = await new Promise((resolve) => {
        command.parse(["info", "--help"], (_err, _argv, output) => {
          console.log("output", output);
          resolve(output);
        });
      });

      expect(output).toContain(FIRST_COMMAND);
      expect(output).toContain(SECOND_COMMAND);
      expect(output).toContain(THIRD_COMMAND);
    });
  });
});
