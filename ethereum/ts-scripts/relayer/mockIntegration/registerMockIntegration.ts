import {
  init,
  loadChains,
  getMockIntegrationAddress,
  getOperatingChains,
} from "../helpers/env";
import { tryNativeToHexString } from "@certusone/wormhole-sdk";
import { XAddressStruct } from "../../../ethers-contracts/MockRelayerIntegration";
import { printRegistration, registerMockIntegration } from "./mockIntegrationDeploy";

const processName = "registerMockIntegration";
init();
const allChains = loadChains();
const operatingChains = getOperatingChains();

async function run() {
  console.log(`Start! ${processName}`);

  const emitters = allChains.map((chain) => ({
    chainId: chain.chainId,
    addr:
      "0x" + tryNativeToHexString(getMockIntegrationAddress(chain), "ethereum"),
  })) satisfies XAddressStruct[];

  const results = await Promise.allSettled(
    operatingChains.map(async (chain) =>
      registerMockIntegration(chain, emitters),
    ),
  );

  for (const result of results) {
    if (result.status === "rejected") {
      console.log(
        `Price update failed: ${result.reason?.stack || result.reason}`,
      );
    } else {
      printRegistration(result.value.updateEmitters, result.value.chain);
    }
  }
}

run().then(() => console.log(`Done! ${processName}`));
