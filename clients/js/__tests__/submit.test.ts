import { describe, expect, it } from "@jest/globals";
import { run_worm_command } from "./utils/cli";
import { getChains, getRpcEndpoint } from "./utils/getters";
import { ContractUpgrade } from "../src/vaa";

const getContractUpgradeVaaByChain = (chain: string) => {
  //TODO: use 'chain' arg to map around future vaa mocks data
  return "sample-vaa";
};

describe("worm submit", () => {
  const chains = getChains();

  describe("check 'ContractUpgrade' functionality", () => {
    chains.forEach((chain) => {
      it.skip(`should submit 'ContractUpgrade' VAA correctly on ${chain}`, async () => {
        const contractUpgradeVAA = getContractUpgradeVaaByChain(chain);
        // Check only 'mainnet' contracts, testnet environments may be unstable
        const rpc = getRpcEndpoint(chain, "MAINNET");
        const network = "mainnet";

        run_worm_command(
          `submit ${contractUpgradeVAA} --chain ${chain} --rpc ${rpc} --network ${network}`
        );

        //TODO: pick up network call and verify its content against expected templates
        const capturedNetworkCall = "mock net call";
        const template = "template of expected net call";

        expect(template).toContain(capturedNetworkCall);
      });
    });
  });
});
