import {
  describe,
  expect,
  it,
  beforeAll,
  afterAll,
  afterEach,
} from "@jest/globals";
import { getRpcEndpoint } from "./utils/getters";
import { server as mswServer, requests as mswRequests } from "./utils/msw";
import yargs from "yargs";
import * as submitCommand from "../src/cmds/submit";
import { YargsCommandModule } from "../src/cmds/Yargs";
import { run_worm_command } from "./utils/cli";

describe("worm submit", () => {
  // Listen to msw local server, network calls are captured there
  beforeAll(() => mswServer.listen());
  afterAll(() => mswServer.close());

  const contractUpgradeModules = [
    "Core",
    "NFTBridge",
    "TokenBridge",
    "WormholeRelayer",
  ];

  describe("check 'ContractUpgrade' functionality", () => {
    // Clean server handlers and request for every test
    afterEach(() => {
      mswServer.resetHandlers();
      mswRequests.length = 0;
    });

    describe("solana", () => {
      const chain = "solana";
      const rpc = getRpcEndpoint(chain, "TESTNET");
      const network = "testnet";

      it(`should submit 'ContractUpgrade' VAA for 'TokenBridge' module correctly on ${chain}`, async () => {
        const module = "TokenBridge";

        //NOTE: use worm generate command to obtain a VAA
        const vaa = run_worm_command(
          `generate upgrade -c ${chain} -m ${module} -a 3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5 -g 0xA240c0e8997D10D59690Cd6Eb36dd55B29af59ACaaa`
        );

        console.log("vaa", vaa);

        await yargs
          .command(submitCommand as unknown as YargsCommandModule)
          .parse(
            `submit ${vaa} --chain ${chain} --rpc ${rpc} --network ${network}`
          );

        expect(mswRequests.length).toBe(1);
        expect(mswRequests[0].url.href).toBe(rpc);
      });
    });
  });
});
