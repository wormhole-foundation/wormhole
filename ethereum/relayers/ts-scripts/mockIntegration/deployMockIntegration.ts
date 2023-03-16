import {
  init,
  loadChains,
  writeOutputFiles,
  getMockIntegration,
  getOperatingChains,
} from "../helpers/env";
import { deployMockIntegration } from "../helpers/deployments";
import { BigNumber } from "ethers";
import { tryNativeToHexString } from "@certusone/wormhole-sdk";
import { MockRelayerIntegration__factory } from "../../sdk/src";

const processName = "deployMockIntegration";
init();
const chains = getOperatingChains();

async function run() {
  console.log("Start! " + processName);
  const output: any = {
    mockIntegrations: [],
  };

  for (let i = 0; i < chains.length; i++) {
    const mockIntegration = await deployMockIntegration(chains[i]);

    output.mockIntegrations.push(mockIntegration);
  }

  writeOutputFiles(output, processName);

  for (let i = 0; i < chains.length; i++) {
    const mockIntegration = getMockIntegration(chains[i]);
    for (let j = 0; j < chains.length; j++) {
      const secondMockIntegration = output.mockIntegrations[j];
      await mockIntegration
        .registerEmitter(
          secondMockIntegration.chainId,
          "0x" +
            tryNativeToHexString(secondMockIntegration.address, "ethereum"),
          { gasLimit: 500000 }
        )
        .then((tx) => tx.wait);
    }
  }
}

run().then(() => console.log("Done!" + processName));
