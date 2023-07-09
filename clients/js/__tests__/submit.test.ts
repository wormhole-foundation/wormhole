import { describe, expect, it } from "@jest/globals";
import { run_worm_command } from "./utils/cli";
import { getChains } from "./utils/getters";

const chains = getChains();

describe("worm submit", () => {
  describe("check functionality", () => {
    chains.forEach((chain) => {
      it(`should submit correctly on ${chain}`, async () => {
        const vaa = "mock vaa";
        run_worm_command(`submit ${vaa}`);

        //TODO: pick up network call and verify its content against expected templates
        const capturedNetworkCall = "mock net call";
        const template = "template of expected net call";

        expect(template).toContain(capturedNetworkCall);
      });
    });
  });
});
