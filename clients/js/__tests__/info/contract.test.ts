import { describe, expect, it } from "@jest/globals";
import { run_worm_command } from "../utils/cli";
import { test_command_positional_args } from "../utils/tests";
import {
  CONTRACTS,
  Network,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { CONTRACT_NOT_DEPLOYED, YARGS_COMMAND_FAILED } from "../utils/errors";
import { getChains, getNetworks } from "../utils/getters";

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
            //TODO: remove 'if statement' once fix on wormhole SDK is merged & published (missing aptos testnet NFTBridge contract as consts)
            // PR source: https://github.com/wormhole-foundation/wormhole/pull/3110
            if (chain === "aptos" && network === "testnet") {
              expect(true).toBe(true);
              return;
            }

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

  describe("check failures", () => {
    it(`should fail if network does not exist`, async () => {
      const fakeNetwork = "DoesNotExist";
      try {
        run_worm_command(`info contract ${fakeNetwork} solana Core`);
      } catch (error) {
        expect((error as Error).message).toContain(YARGS_COMMAND_FAILED);
      }
    });

    it(`should fail if chain does not exist`, async () => {
      const fakeChain = "DoesNotExist";
      try {
        run_worm_command(`info contract mainnet ${fakeChain} Core`);
      } catch (error) {
        expect((error as Error).message).toContain(YARGS_COMMAND_FAILED);
      }
    });

    it(`should fail if module (Core, NFTBridge, TokenBridge) does not exist`, async () => {
      const fakeModule = "DoesNotExist";
      try {
        run_worm_command(`info contract mainnet solana ${fakeModule}`);
      } catch (error) {
        expect((error as Error).message).toContain(YARGS_COMMAND_FAILED);
      }
    });
  });
});
