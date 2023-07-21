import {
  describe,
  expect,
  it,
  beforeAll,
  afterAll,
  afterEach,
  jest,
} from "@jest/globals";
import { WormholeSDKChainName, getRpcEndpoint } from "./utils/getters";
import { server as mswServer, requests } from "./utils/msw";
import yargs from "yargs";
import * as submitCommand from "../src/cmds/submit";
import { YargsCommandModule } from "../src/cmds/Yargs";
import { run_worm_command } from "./utils/cli";
import { INVALID_VAA_CHAIN } from "./utils/errors";

describe("worm submit", () => {
  let originalProcessExit: any;

  beforeAll(() => {
    // Save original process.exit, needed to recover exited processes
    originalProcessExit = process.exit;
    // Override process.exit
    process.exit = jest.fn(() => {
      throw new Error("process.exit was called");
    });
    // Listen to msw local server, network calls are captured there
    mswServer.listen();
  });
  afterAll(() => {
    process.exit = originalProcessExit;
    mswServer.close();
  });

  const contractUpgradeModules = ["Core", "NFTBridge", "TokenBridge"];
  const mockGuardianAddress = "0xA240c0e8997D10D59690Cd6Eb36dd55B29af59ACaaa";

  describe("check 'ContractUpgrade' functionality", () => {
    // Clean server handlers and request for every test
    afterEach(() => {
      mswServer.resetHandlers();
      requests.length = 0;
    });
    const testTimeout = 10000;

    describe.only("solana", () => {
      const chain: WormholeSDKChainName = "solana";
      const rpc = getRpcEndpoint(chain, "TESTNET"); // generated vaa from 'worm generate' command does not work on mainnet, use testnet instead
      const network = "testnet";

      contractUpgradeModules.forEach((module) => {
        it(
          `should send transaction to ${chain} when submitting 'ContractUpgrade' VAA for '${module}' module`,
          async () => {
            //NOTE: use worm generate command to obtain a VAA
            const vaa = run_worm_command(
              `generate upgrade -c ${chain} -m ${module} -a 3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5 -g ${mockGuardianAddress}`
            );

            //NOTE: we capture requests sent, then we force this process to fail before sending transactions
            try {
              await yargs
                .command(submitCommand as unknown as YargsCommandModule)
                .parse(
                  `submit ${vaa} --chain ${chain} --rpc ${rpc} --network ${network}`
                );
            } catch (e) {}

            expect(requests.length).toBe(7);
            expect(
              requests.some((req) => req.body.method === "sendTransaction")
            ).toBeTruthy();
          },
          testTimeout
        );
      });

      it(
        `should fail to send transactions to ${chain} when submitting 'ContractUpgrade' VAA, if 'vaa' is malformed`,
        async () => {
          const fakeVaa = "this-is-a-fake-vaa";
          try {
            //NOTE: we capture requests sent, then we force this process to fail before sending transactions
            await yargs
              .command(submitCommand as unknown as YargsCommandModule)
              .parse(
                `submit ${fakeVaa} --chain ${chain} --rpc ${rpc} --network ${network}`
              );
          } catch (e) {}

          expect(requests.length).toBe(0);
        },
        testTimeout
      );

      it(
        `should throw error if chain defined in 'vaa' is different than target chain (${chain})`,
        async () => {
          //NOTE: use worm generate command to obtain a VAA from a different chain (ethereum)
          const vaaFromOtherChain = run_worm_command(
            `generate upgrade -c ethereum -m Core -a 0xF890982f9310df57d00f659cf4fd87e65adEd8d7 -g ${mockGuardianAddress}`
          );
          try {
            //NOTE: we capture requests sent, then we force this process to fail before sending transactions
            await yargs
              .command(submitCommand as unknown as YargsCommandModule)
              .parse(
                `submit ${vaaFromOtherChain} --chain ${chain} --rpc ${rpc} --network ${network}`
              );
          } catch (error) {
            expect(String(error)).toBe(INVALID_VAA_CHAIN(chain, "ethereum"));
          }
        },
        testTimeout
      );
    });

    describe("evm", () => {
      const evmChains: WormholeSDKChainName[] = [
        "ethereum",
        "arbitrum",
        "aurora",
        "avalanche",
        "bsc",
        "celo",
        "fantom",
        "gnosis",
        "klaytn",
        "moonbeam",
        "oasis",
        "optimism",
        "polygon",
      ];

      evmChains.forEach((chain) => {
        describe(`${chain}`, () => {
          const rpc = getRpcEndpoint(chain, "MAINNET");
          const network = "mainnet";

          contractUpgradeModules.forEach((module) => {
            if (chain === "gnosis" && module !== "Core") return; // Handle special case for 'gnosis' chain, it only has 'Core' contract

            it(
              `should send transaction to ${chain} when submitting 'ContractUpgrade' VAA for '${module}' module`,
              async () => {
                //NOTE: use worm generate command to obtain a VAA
                const vaa = run_worm_command(
                  `generate upgrade -c ${chain} -m ${module} -a 0xF890982f9310df57d00f659cf4fd87e65adEd8d7 -g ${mockGuardianAddress}`
                );

                //NOTE: we capture requests sent, then we force this process to fail before sending transactions
                try {
                  await yargs
                    .command(submitCommand as unknown as YargsCommandModule)
                    .parse(
                      `submit ${vaa} --chain ${chain} --rpc ${rpc} --network ${network}`
                    );
                } catch (e) {}

                expect(
                  requests.some(
                    (req) => req.body.method === "eth_sendRawTransaction"
                  )
                ).toBeTruthy();
              },
              testTimeout
            );
          });

          it(
            `should fail to send transactions to ${chain} when submitting 'ContractUpgrade' VAA, if 'vaa' is malformed`,
            async () => {
              const fakeVaa = "this-is-a-fake-vaa";
              try {
                //NOTE: we capture requests sent, then we force this process to fail before sending transactions
                await yargs
                  .command(submitCommand as unknown as YargsCommandModule)
                  .parse(
                    `submit ${fakeVaa} --chain ${chain} --rpc ${rpc} --network ${network}`
                  );
              } catch (e) {}

              expect(requests.length).toBe(0);
            },
            testTimeout
          );

          it(
            `should throw error if chain defined in 'vaa' is different than target chain (${chain})`,
            async () => {
              //NOTE: use worm generate command to obtain a VAA from a different chain (solana)
              const vaaFromOtherChain = run_worm_command(
                `generate upgrade -c solana -m Core -a 3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5 -g ${mockGuardianAddress}`
              );
              try {
                //NOTE: we capture requests sent, then we force this process to fail before sending transactions
                await yargs
                  .command(submitCommand as unknown as YargsCommandModule)
                  .parse(
                    `submit ${vaaFromOtherChain} --chain ${chain} --rpc ${rpc} --network ${network}`
                  );
              } catch (error) {
                expect(String(error)).toBe(INVALID_VAA_CHAIN(chain, "solana"));
              }
            },
            testTimeout
          );
        });
      });
    });
  });
});
