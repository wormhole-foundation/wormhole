import {
  init,
  loadChains,
  writeOutputFiles,
  getMockIntegration,
  Deployment,
  getOperatingChains,
  getMockIntegrationAddress,
} from "../helpers/env";
import { deployMockIntegration, buildOverrides } from "../helpers/deployments";
import type { BigNumberish, BytesLike } from "ethers";
import { tryNativeToHexString } from "@certusone/wormhole-sdk";
import { wait } from "../helpers/utils";

const processName = "deployMockIntegration";
init();
const chains = loadChains();
const operatingChains = getOperatingChains();

async function run() {
  console.log("Start!");
  const output = {
    mockIntegrations: [] as Deployment[],
  };

  for (const chain of operatingChains) {
    const mockIntegration = await deployMockIntegration(chain);
    output.mockIntegrations.push(mockIntegration);
  }

  writeOutputFiles(output, processName);

  for (const chain of operatingChains) {
    console.log(`Registering emitters for chainId ${chain.chainId}`);
    // note: must use useLastRun = true
    const mockIntegration = await getMockIntegration(chain);

    const emitters: {
      chainId: BigNumberish;
      addr: BytesLike;
    }[] = chains.map((c, j) => ({
      chainId: c.chainId,
      addr:
        "0x" + tryNativeToHexString(getMockIntegrationAddress(c), "ethereum"),
    }));

    const overrides = await buildOverrides(
      () => mockIntegration.estimateGas.registerEmitters(emitters),
      chain
    );
    await mockIntegration.registerEmitters(emitters, overrides).then(wait);
  }
}

run().then(() => console.log("Done!"));
