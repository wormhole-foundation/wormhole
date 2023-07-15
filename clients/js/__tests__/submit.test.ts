import { describe, expect, it } from "@jest/globals";
import { run_worm_command } from "./utils/cli";
import {
  WormholeSDKChainName,
  getChains,
  getRpcEndpoint,
} from "./utils/getters";
import { ContractUpgrade } from "../src/vaa";

const getContractUpgradeVaaByChain = (chain: string) => {
  //TODO: use 'chain' arg to map around future vaa mocks data
  if (chain === "solana") {
    const solanaUpgradeVaaTokenContract =
      "01000000000200bca7dda78ef9fe96ef829959850fc5b02b49eeb839697657b3e2f477f75f4f204a313ab3de929ae26d95759b1be478050156747f506835c9b65a487eea0882420101de279fb9987c5c095ae960118f9c28bd122d0354119c7c9112d0410eaa93f10f6d612d87684ead5f7c6b4335ee77f550b0d93b867e14c1dee1aef90fd34db71b000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000002df553400000000000000000000000000000000000000000000546f6b656e4272696467650200010e0a589e6488147a94dcfa592b90fdd41152bb2ca77bf6016758a6f4df9d21b4";

    return solanaUpgradeVaaTokenContract;
  }
};

describe("worm submit", () => {
  const chains = ["solana"] as WormholeSDKChainName[];

  describe("check 'ContractUpgrade' functionality", () => {
    chains.forEach((chain) => {
      it.skip(`should submit 'ContractUpgrade' VAA correctly on ${chain}`, async () => {
        const contractUpgradeVAA = getContractUpgradeVaaByChain(chain);
        // Check only 'mainnet' contracts, testnet environments may be unstable
        const rpc = getRpcEndpoint(chain, "MAINNET");
        const network = "mainnet";

        run_worm_command(
          `submit ${contractUpgradeVAA} --chain ${chain} --rpc ${rpc} --network ${network}`
        );

        //TODO: pick up network call and verify its content against expected templates
        const capturedNetworkCall = "mock net call";
        const template = "template of expected net call";

        expect(template).toContain(capturedNetworkCall);
      });
    });
  });
});
