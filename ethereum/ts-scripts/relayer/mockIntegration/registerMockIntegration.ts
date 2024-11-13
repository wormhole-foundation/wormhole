import {
  init,
  loadChains,
  getMockIntegrationAddress,
  getOperatingChains,
} from "../helpers/env";
import { XAddressStruct } from "../../../ethers-contracts/MockRelayerIntegration";
import { printRegistration, registerMockIntegration } from "./mockIntegrationDeploy";
import { nativeEthereumAddressToHex } from "../helpers/utils";

const processName = "registerMockIntegration";
init();
const allChains = loadChains();
const operatingChains = getOperatingChains();

async function run() {
  console.log(`Start! ${processName}`);

  const emitters = allChains.map((chain) => ({
    chainId: chain.chainId,
    addr: nativeEthereumAddressToHex(getMockIntegrationAddress(chain)),
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
