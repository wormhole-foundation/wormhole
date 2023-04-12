import yargs from "yargs";
import { describe, expect, it, jest } from "@jest/globals";

describe("worm recover", () => {
  describe("check arguments", () => {
    const FIRST_POSITIONAL_ARGUMENT = "<digest>";
    const SECOND_POSITIONAL_ARGUMENT = "<signature>";

    it(`should have correct positional arguments`, async () => {
      const command = await yargs.command(require("../cmds/recover")).help();

      // Run the command module with --help as argument
      const output = await new Promise((resolve) => {
        command.parse(["recover", "--help"], (_err, _argv, output) => {
          console.log("output", output);
          resolve(output);
        });
      });

      expect(output).toContain(FIRST_POSITIONAL_ARGUMENT);
      expect(output).toContain(SECOND_POSITIONAL_ARGUMENT);
    });
  });

  describe("check functionality", () => {
    const MOCK_DIGEST =
      "0x99656f88302bda18573212d4812daeea7d39f8af695db1fbc4d99fd94f552606";
    const MOCK_SIGNATURE =
      "6da03c5e56cb15aeeceadc1e17a45753ab4dc0ec7bf6a75ca03143ed4a294f6f61bc3f478a457833e43084ecd7c985bf2f55a55f168aac0e030fc49e845e497101";

    const EXPECTED_ADDRESS = "0x6FbEBc898F403E4773E95feB15E80C9A99c8348d";

    it(`should return correct address from mock digest and signature`, async () => {
      const consoleSpy = jest.spyOn(console, "log");

      const command = yargs.command(require("../cmds/recover")).help();
      await command.parse(["recover", MOCK_DIGEST, MOCK_SIGNATURE]);

      expect(consoleSpy).toBeCalledWith(EXPECTED_ADDRESS);
    });
  });
});
