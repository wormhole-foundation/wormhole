import { describe, expect, it } from "@jest/globals";
import { run_worm_command } from "../utils/cli";
import { test_command_positional_args } from "../utils/tests";
import { NETWORKS as RPC_NETWORKS } from "../../src/consts/networks";
import { getChains, getNetworks } from "../utils/getters";
import { Network } from "@certusone/wormhole-sdk/lib/esm/utils/consts";

describe("worm info rpc", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["network", "chain"];

    test_command_positional_args("info rpc", args);
  });

  describe("check functionality", () => {
    const chains = getChains();
    const networks = getNetworks();

    networks.forEach((network) => {
      const NETWORK = network.toUpperCase() as Network;

      chains.forEach((chain) => {
        it(`should return ${chain} ${network} rpc correctly`, async () => {
          const output = run_worm_command(`info rpc ${network} ${chain}`);

          expect(output).toContain(String(RPC_NETWORKS[NETWORK][chain].rpc));
        });
      });
    });
  });
});
