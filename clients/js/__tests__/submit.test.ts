import {
  describe,
  expect,
  it,
  beforeAll,
  afterAll,
  afterEach,
} from "@jest/globals";
import { WormholeSDKChainName, getRpcEndpoint } from "./utils/getters";
import { server as mswServer, requests as mswRequests } from "./utils/msw";
import yargs from "yargs";
import * as submitCommand from "../src/cmds/submit";
import { YargsCommandModule } from "../src/cmds/Yargs";

const getContractUpgradeVaaByChain = (chain: string) => {
  //TODO: use 'chain' arg to map around future vaa mocks data
  if (chain === "solana") {
    const solanaUpgradeVaaTokenContract =
      "01000000000200c2196789ccee7ce30e300e09626b2bf594729ef17b7f6b627308527c0f5929a501cf48f10f141717b299e79cc72ab4581c0fd8e23c8fe5de40b9a13d8f6e3ce90001c2196789ccee7ce30e300e09626b2bf594729ef17b7f6b627308527c0f5929a501cf48f10f141717b299e79cc72ab4581c0fd8e23c8fe5de40b9a13d8f6e3ce90000000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000025124a40000000000000000000000000000000000000000000000000000000000436f72650100012b1246c9eefa3c466792253111f35fec1ee8ee5e9debc412d2e9adadfecdcc72";
    return solanaUpgradeVaaTokenContract;
  }
};

describe("worm submit", () => {
  const chains = ["solana"] as WormholeSDKChainName[];

  // Listen to msw local server, network calls are captured there
  beforeAll(() => mswServer.listen());
  afterAll(() => mswServer.close());

  describe("check 'ContractUpgrade' functionality", () => {
    // Clean server handlers and request for every test
    afterEach(() => {
      mswServer.resetHandlers();
      mswRequests.length = 0;
    });

    chains.forEach((chain) => {
      it(`should submit 'ContractUpgrade' VAA for 'TokenBridge' module correctly on ${chain}`, async () => {
        const contractUpgradeVAA = getContractUpgradeVaaByChain(chain);
        // Check only 'mainnet' contracts, testnet environments may be unstable
        const rpc = getRpcEndpoint(chain, "MAINNET");
        const network = "testnet";

        const argv = await yargs
          .command(submitCommand as unknown as YargsCommandModule)
          .parse(
            `submit ${contractUpgradeVAA} --chain ${chain} --rpc ${rpc} --network ${network}`
          );

        expect(mswRequests.length).toBe(1);
        expect(mswRequests[0].url.href).toBe(rpc);
      });
    });
  });
});
