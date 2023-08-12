import {
  init,
  loadChains,
  writeOutputFiles,
  getMockIntegration,
  Deployment,
  loadLastRun,
  getOperationDescriptor,
  getMockIntegrationAddress,
} from "../helpers/env";
import { deployMockIntegration, buildOverrides } from "../helpers/deployments";
import type { BigNumberish, BytesLike } from "ethers";
import { tryNativeToHexString } from "@certusone/wormhole-sdk";
import { wait } from "../helpers/utils";

const processName = "deployMockIntegration";
init();
const chains = loadChains();
const operation = getOperationDescriptor();

interface MockIntegrationDeployment {
  mockIntegrations: Deployment[];
}

async function run() {
  console.log("Start!");

  const lastRun: MockIntegrationDeployment | undefined =
    loadLastRun(processName);
  const output = {
    mockIntegrations: lastRun?.mockIntegrations?.filter(isSupportedChain) || [],
  };

  for (const chain of operation.operatingChains) {
    const mockIntegration = await deployMockIntegration(chain);
    output.mockIntegrations.push(mockIntegration);
  }

  writeOutputFiles(output, processName);

  for (const chain of operation.operatingChains) {
    console.log(`Registering emitters for chainId ${chain.chainId}`);
    // note: must use useLastRun = true
    const mockIntegration = await getMockIntegration(chain);

    const emitters: {
      chainId: BigNumberish;
      addr: BytesLike;
    }[] = chains.map((c) => ({
      chainId: c.chainId,
      addr:
        "0x" + tryNativeToHexString(getMockIntegrationAddress(c), "ethereum"),
    }));

    const overrides = await buildOverrides(
      () => mockIntegration.estimateGas.registerEmitters(emitters),
      chain,
    );
    await mockIntegration.registerEmitters(emitters, overrides).then(wait);
  }
}

function isSupportedChain(deploy: Deployment): boolean {
  const item = operation.supportedChains.find((chain) => {
    return deploy.chainId === chain.chainId;
  });
  return item !== undefined;
}

run().then(() => console.log("Done!"));
