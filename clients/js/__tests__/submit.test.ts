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

const runFailureCases = (
  chain: WormholeSDKChainName,
  rpc: string,
  network: string,
  mockGuardianAddress: string,
  vaaChain: WormholeSDKChainName = "solana",
  vaaAddress: string = "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5"
) => {
  const testTimeout = 10000;

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
      //NOTE: use worm generate command to obtain a VAA from a different chain
      const vaaFromOtherChain = run_worm_command(
        `generate upgrade -c ${vaaChain} -m Core -a ${vaaAddress} -g ${mockGuardianAddress}`
      );
      try {
        //NOTE: we capture requests sent, then we force this process to fail before sending transactions
        await yargs
          .command(submitCommand as unknown as YargsCommandModule)
          .parse(
            `submit ${vaaFromOtherChain} --chain ${chain} --rpc ${rpc} --network ${network}`
          );
      } catch (error) {
        expect(String(error)).toBe(INVALID_VAA_CHAIN(chain, `${vaaChain}`));
      }
    },
    testTimeout
  );
};

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

    describe("solana", () => {
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

            expect(
              requests.some((req) => req.body.method === "sendTransaction")
            ).toBeTruthy();
          },
          testTimeout
        );
      });

      runFailureCases(
        chain,
        rpc,
        network,
        mockGuardianAddress,
        "ethereum",
        "0xF890982f9310df57d00f659cf4fd87e65adEd8d7"
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

          runFailureCases(chain, rpc, network, mockGuardianAddress);
        });
      });
    });

    describe("aptos", () => {
      const chain: WormholeSDKChainName = "aptos";
      const rpc = getRpcEndpoint(chain, "MAINNET");
      const network = "mainnet";

      contractUpgradeModules.forEach((module) => {
        it(
          `should send transaction to ${chain} when submitting 'ContractUpgrade' VAA for '${module}' module`,
          async () => {
            //NOTE: use worm generate command to obtain a VAA
            const vaa = run_worm_command(
              `generate upgrade -c ${chain} -m ${module} -a 0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625 -g ${mockGuardianAddress}`
            );

            //NOTE: we capture requests sent, then we force this process to fail before sending transactions
            try {
              await yargs
                .command(submitCommand as unknown as YargsCommandModule)
                .parse(
                  `submit ${vaa} --chain ${chain} --rpc ${rpc} --network ${network}`
                );
            } catch (e) {}

            // We expect it to perform a transaction simulation
            expect(
              requests.some((req) =>
                req.url.pathname.includes("/transactions/simulate")
              )
            ).toBeTruthy();

            // We expect it to launch the actual transaction
            // Using regex, as previous call '/transactions/simulate' can provide us a false positive with '/transactions' call
            expect(
              requests.some((req) =>
                new RegExp(/\/transactions$/).test(req.url.pathname)
              )
            ).toBeTruthy();
          },
          testTimeout
        );
      });

      runFailureCases(chain, rpc, network, mockGuardianAddress);
    });

    describe("sui", () => {
      const chain: WormholeSDKChainName = "sui";
      const rpc = getRpcEndpoint(chain, "MAINNET");
      const network = "mainnet";

      it(
        `should return error 'ContractUpgrade not supported on Sui'`,
        async () => {
          //NOTE: use worm generate command to obtain a VAA
          const vaa = run_worm_command(
            `generate upgrade -c ${chain} -m TokenBridge -a 0xaeab97f96cf9877fee2883315d459552b2b921edc16d7ceac6eab944dd88919c -g ${mockGuardianAddress}`
          );

          try {
            await yargs
              .command(submitCommand as unknown as YargsCommandModule)
              .parse(
                `submit ${vaa} --chain ${chain} --rpc ${rpc} --network ${network}`
              );
          } catch (error) {
            expect(String(error)).toBe(
              "Error: ContractUpgrade not supported on Sui"
            );
          }
        },
        testTimeout
      );
    });

    describe("near", () => {
      const chain: WormholeSDKChainName = "near";
      const rpc = getRpcEndpoint(chain, "MAINNET");
      const network = "mainnet";

      contractUpgradeModules.forEach((module) => {
        // NEAR does not have a current NFTBridge contract on Mainnet. Source: https://docs.wormhole.com/wormhole/reference/environments/near
        if (module === "NFTBridge") return;

        it(
          `should send transaction to ${chain} when submitting 'ContractUpgrade' VAA for '${module}' module`,
          async () => {
            //NOTE: use worm generate command to obtain a VAA
            const vaa = run_worm_command(
              `generate upgrade -c ${chain} -m ${module} -a 0x148410499d3fcda4dcfd68a1ebfcdddda16ab28326448d4aae4d2f0465cdfcb7 -g ${mockGuardianAddress}`
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
              requests.some((req) => req.body.method === "broadcast_tx_commit")
            ).toBeTruthy();
          },
          testTimeout
        );
      });

      runFailureCases(chain, rpc, network, mockGuardianAddress);
    });

    describe("algorand", () => {
      const chain: WormholeSDKChainName = "algorand";
      const rpc = getRpcEndpoint(chain, "MAINNET");
      const network = "mainnet";

      contractUpgradeModules.forEach((module) => {
        // algorand does not have a current NFTBridge contract on Mainnet. Source: https://docs.wormhole.com/wormhole/reference/environments/algorand
        if (module === "NFTBridge") return;

        it(`should send transaction to ${chain} when submitting 'ContractUpgrade' VAA for '${module}' module`, async () => {
          //NOTE: use worm generate command to obtain a VAA
          const vaa = run_worm_command(
            `generate upgrade -c ${chain} -m ${module} -a 0x67e93fa6c8ac5c819990aa7340c0c16b508abb1178be9b30d024b8ac25193d45 -g ${mockGuardianAddress}`
          );

          //NOTE: we capture requests sent, then we force this process to fail before sending transactions
          try {
            await yargs
              .command(submitCommand as unknown as YargsCommandModule)
              .parse(
                `submit ${vaa} --chain ${chain} --rpc ${rpc} --network ${network}`
              );
          } catch (e) {}

          // We expect it to perform a transaction
          expect(
            requests.some(
              (req) =>
                req.url.pathname.includes("/transactions") &&
                req.method === "POST" &&
                !!req.body // verify body has transaction data to send
            )
          ).toBeTruthy();
        }, 30000); // algorand needs more time to fail, as it retries API call several times
      });

      runFailureCases(chain, rpc, network, mockGuardianAddress);
    });
  });
});
