import { describe, expect, it } from "@jest/globals";
import { run_worm_command } from "../utils/cli";
import { test_command_positional_args } from "../utils/tests";
import { CHAINS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { YARGS_COMMAND_FAILED } from "../utils/errors";
import { WormholeSDKChainName, getChains } from "../utils/getters";

describe("worm info chain-id", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["chain"];

    test_command_positional_args("info chain-id", args);
  });

  describe("check functionality", () => {
    const chains: WormholeSDKChainName[] = getChains();

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
