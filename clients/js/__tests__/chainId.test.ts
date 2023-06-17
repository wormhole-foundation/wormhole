import { describe, expect, it } from "@jest/globals";
import { run_worm_command, run_worm_help_command } from "./utils-jest";
import { CHAINS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { YARGS_COMMAND_FAILED } from "./yargs-errors";

describe("worm info chain-id", () => {
  describe("check arguments", () => {
    const args = ["chain"];

    it(`should have correct positional arguments`, async () => {
      // Run the command module with --help as argument
      const output = run_worm_help_command("info chain-id");

      args.forEach((arg) => {
        expect(output).toContain(`<${arg}>`);
      });
    });
  });

  describe("check functionality", () => {
    type WormholeSDKChainName = keyof typeof CHAINS;

    const chains = Object.keys(CHAINS) as WormholeSDKChainName[];

    chains.forEach((chain) => {
      it(`should return ${chain} chain-id correctly`, async () => {
        const output = run_worm_command(`info chain-id ${chain}`);
        expect(output).toContain(CHAINS[chain].toString());
      });
    });
  });

  describe("check failures", () => {
    it(`should fail if chain does not exist`, async () => {
      const fakeChain = "IDontExist";
      try {
        run_worm_command(`info chain-id ${fakeChain}`);
      } catch (error) {
        expect((error as Error).message).toContain(YARGS_COMMAND_FAILED);
      }
    });
  });
});
