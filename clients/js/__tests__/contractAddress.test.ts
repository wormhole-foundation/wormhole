import yargs from "yargs";
import { describe, expect, it } from "@jest/globals";
import { run_worm_command, test_command_positional_args } from "./utils-jest";
import { CONTRACTS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { WormholeSDKChainName, getChains } from "./chain-id.test";
import { CORE_CONTRACT_NOT_DEPLOYED } from "./errors";

describe("worm info contract", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["network", "chain", "module"];

    test_command_positional_args("info contract", args);
  });

  describe("check functionality", () => {
    const chains: WormholeSDKChainName[] = getChains();

    describe("should return core mainnet contracts", () => {
      chains.forEach((chain) => {
        it(`should return ${chain} core mainnet contract correctly`, async () => {
          try {
            const output = run_worm_command(
              `info contract mainnet ${chain} Core`
            );
            expect(output).toEqual(CONTRACTS["MAINNET"][chain]["core"]);
          } catch (error) {
            expect((error as Error).message).toContain(
              CORE_CONTRACT_NOT_DEPLOYED(chain)
            );
          }
        });
      });
    });

    // it.skip(`should return solana core mainnet contract correctly`, async () => {
    //   const consoleSpy = jest.spyOn(console, "log");

    //   const command = yargs
    //     .command(require("../src/cmds/contractAddress"))
    //     .help();
    //   await command.parse(["contract", "mainnet", "solana", "Core"]);

    //   expect(consoleSpy).toBeCalledWith(SOLANA_CORE_CONTRACT);
    // });

    // it.skip(`should return ethereum mainnet NFTBridge contract correctly`, async () => {
    //   const consoleSpy = jest.spyOn(console, "log");

    //   const command = yargs
    //     .command(require("../src/cmds/contractAddress"))
    //     .help();
    //   await command.parse(["contract", "mainnet", "ethereum", "NFTBridge"]);

    //   expect(consoleSpy).toBeCalledWith(ETHEREUM_NFT_BRIDGE_CONTRACT);
    // });
  });
});
