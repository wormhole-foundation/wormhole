import {
  describe,
  expect,
  it,
  beforeAll,
  afterAll,
  afterEach,
  jest,
} from "@jest/globals";
import { getRpcEndpoint } from "./utils/getters";
import { server as mswServer, requests } from "./utils/msw";
import yargs from "yargs";
import * as submitCommand from "../src/cmds/submit";
import { YargsCommandModule } from "../src/cmds/Yargs";
import { run_worm_command } from "./utils/cli";

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

  describe("check 'ContractUpgrade' functionality", () => {
    // Clean server handlers and request for every test
    afterEach(() => {
      mswServer.resetHandlers();
      requests.length = 0;
    });
    const testTimeout = 10000;

    describe("solana", () => {
      const chain = "solana";
      const rpc = getRpcEndpoint(chain, "TESTNET");
      const network = "testnet";

      contractUpgradeModules.forEach((module) => {
        it(
          `should send transaction when submitting 'ContractUpgrade' VAA for '${module}' module on ${chain}`,
          async () => {
            //NOTE: use worm generate command to obtain a VAA
            const vaa = run_worm_command(
              `generate upgrade -c ${chain} -m ${module} -a 3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5 -g 0xA240c0e8997D10D59690Cd6Eb36dd55B29af59ACaaa`
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
        `should fail to send transactions when submitting 'ContractUpgrade' VAA on ${chain}, if 'vaa' is malformed`,
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
    });
  });
});
