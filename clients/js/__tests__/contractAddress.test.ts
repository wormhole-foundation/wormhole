import { describe, expect, it } from "@jest/globals";
import { run_worm_command, test_command_positional_args } from "./utils-cli";
import {
  CONTRACTS,
  Network,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { CONTRACT_NOT_DEPLOYED } from "./errors";
import { getChains, getNetworks } from "./utils";

describe("worm info contract", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["network", "chain", "module"];

    test_command_positional_args("info contract", args);
  });

  describe("check functionality", () => {
    const chains = getChains();
    const networks = getNetworks();

    networks.forEach((network) => {
      const NETWORK = network.toUpperCase() as Network;

      describe(`should return ${network} contracts`, () => {
        chains.forEach((chain) => {
          it(`should return ${chain} core ${network} contract correctly`, async () => {
            try {
              const output = run_worm_command(
                `info contract ${network} ${chain} Core`
              );
              expect(output).toContain(CONTRACTS[NETWORK][chain]["core"]);
            } catch (error) {
              expect((error as Error).message).toContain(
                CONTRACT_NOT_DEPLOYED(chain, "Core")
              );
            }
          });

          it(`should return ${chain} NFTBridge ${network} contract correctly`, async () => {
            try {
              const output = run_worm_command(
                `info contract ${network} ${chain} NFTBridge`
              );
              expect(output).toContain(CONTRACTS[NETWORK][chain]["nft_bridge"]);
            } catch (error) {
              expect((error as Error).message).toContain(
                CONTRACT_NOT_DEPLOYED(chain, "NFTBridge")
              );
            }
          });

          it(`should return ${chain} TokenBridge ${network} contract correctly`, async () => {
            try {
              const output = run_worm_command(
                `info contract ${network} ${chain} TokenBridge`
              );
              expect(output).toContain(
                CONTRACTS[NETWORK][chain]["token_bridge"]
              );
            } catch (error) {
              expect((error as Error).message).toContain(
                CONTRACT_NOT_DEPLOYED(chain, "TokenBridge")
              );
            }
          });
        });
      });
    });
  });
});
