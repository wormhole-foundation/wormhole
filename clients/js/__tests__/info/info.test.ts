import { describe, expect, it } from "@jest/globals";
import { run_worm_help_command } from "../utils-cli";

describe("worm info", () => {
  describe("check commands", () => {
    const commands = [
      "chain-id",
      "contract",
      "emitter",
      "origin",
      "registrations",
      "rpc",
      "wrapped",
    ];

    const output = run_worm_help_command("info");

    commands.forEach((command) => {
      it(`should have ${command} in worm info namespace`, async () => {
        expect(output).toContain(`worm info ${command}`);
      });
    });
  });
});
