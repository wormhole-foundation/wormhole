import { describe, expect, it } from "@jest/globals";
import { run_worm_command } from "../utils/cli";
import {
  CONTRACTS,
  Network,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { CONTRACT_NOT_DEPLOYED, YARGS_COMMAND_FAILED } from "../utils/errors";
import { getChains, networks } from "../utils/getters";

describe("worm info contract", () => {
  describe("check functionality", () => {
    const chains = getChains();

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