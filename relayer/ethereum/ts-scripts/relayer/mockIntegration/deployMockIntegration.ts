import {
  init,
  loadChains,
  writeOutputFiles,
  getMockIntegration,
  Deployment,
  getOperatingChains,
  getMockIntegrationAddress,
} from "../helpers/env";
import { deployMockIntegration } from "../helpers/deployments";
import { BigNumber, BigNumberish, BytesLike } from "ethers";
import {
  tryNativeToHexString,
  tryNativeToUint8Array,
} from "@certusone/wormhole-sdk";
import { MockRelayerIntegration__factory } from "../../../ethers-contracts";
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

  for (let i = 0; i < operatingChains.length; i++) {
    const mockIntegration = await deployMockIntegration(operatingChains[i]);
    output.mockIntegrations.push(mockIntegration);
  }

  writeOutputFiles(output, processName);

  for (let i = 0; i < operatingChains.length; i++) {
    console.log(
      `Registering emitters for chainId ${operatingChains[i].chainId}`
    );
    // note: must use useLastRun = true
    const mockIntegration = getMockIntegration(operatingChains[i]);

    const arg: {
      chainId: BigNumberish;
      addr: BytesLike;
    }[] = chains.map((c, j) => ({
      chainId: c.chainId,
      addr:
        "0x" + tryNativeToHexString(getMockIntegrationAddress(c), "ethereum"),
    }));

    await mockIntegration
      .registerEmitters(arg, { gasLimit: 500000 })
      .then(wait);
  }
}

run().then(() => console.log("Done!"));
