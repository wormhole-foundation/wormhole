import { describe, expect, it } from "@jest/globals";
import { run_worm_command, test_command_positional_args } from "./utils-cli";
import { CONTRACTS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { CONTRACT_NOT_DEPLOYED } from "./errors";
import { WormholeSDKChainName, getChains } from "./utils";

describe("worm info contract", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["network", "chain", "module"];

    test_command_positional_args("info contract", args);
  });

  describe("check functionality", () => {
    const chains: WormholeSDKChainName[] = getChains();

    describe("should return mainnet contracts", () => {
      chains.forEach((chain) => {
        it(`should return ${chain} core mainnet contract correctly`, async () => {
          try {
            const output = run_worm_command(
              `info contract mainnet ${chain} Core`
            );
            expect(output).toContain(CONTRACTS["MAINNET"][chain]["core"]);
          } catch (error) {
            expect((error as Error).message).toContain(
              CONTRACT_NOT_DEPLOYED(chain, "Core")
            );
          }
        });

        it(`should return ${chain} NFTBridge mainnet contract correctly`, async () => {
          try {
            const output = run_worm_command(
              `info contract mainnet ${chain} NFTBridge`
            );
            expect(output).toContain(CONTRACTS["MAINNET"][chain]["nft_bridge"]);
          } catch (error) {
            expect((error as Error).message).toContain(
              CONTRACT_NOT_DEPLOYED(chain, "NFTBridge")
            );
          }
        });

        it(`should return ${chain} TokenBridge mainnet contract correctly`, async () => {
          try {
            const output = run_worm_command(
              `info contract mainnet ${chain} TokenBridge`
            );
            expect(output).toContain(
              CONTRACTS["MAINNET"][chain]["token_bridge"]
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
