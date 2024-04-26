import {
  init,
  writeOutputFiles,
  Deployment,
  getOperationDescriptor,
  loadMockIntegrations,
  getChain,
  getSigner,
} from "../helpers/env";
import { tryNativeToHexString } from "@certusone/wormhole-sdk";
import { deployMockIntegration, buildOverrides } from "../helpers/deployments";
import { wait } from "../helpers/utils";
import { XAddressStruct } from "../../../ethers-contracts/MockRelayerIntegration";
import { MockRelayerIntegration__factory } from "../../../ethers-contracts";

const processName = "deployMockIntegration";
init();
const operation = getOperationDescriptor();

interface MockIntegrationDeployment {
  mockIntegrations: Deployment[];
}

async function run() {
  console.log("Start!");

  const oldDeployments = loadMockIntegrations().filter(isSupportedChain);
  const newDeployments: Deployment[] = [];

  // TODO: deploy only on chains missing deployment
  const deploymentTasks = await Promise.allSettled(operation.operatingChains.map(async (chain) => {
    return deployMockIntegration(chain);
  }));

  for (const task of deploymentTasks) {
    if (task.status === "rejected") {
      // These get discarded and need to be retried later with a separate invocation.
      console.log(task.reason?.stack || task.reason);
    } else {
      newDeployments.push(task.value);
    }
  }

  const output = {
    mockIntegrations: oldDeployments.concat(newDeployments),
  } satisfies MockIntegrationDeployment;
  writeOutputFiles(output, processName);

  const emitters = output.mockIntegrations.map(({address, chainId}) => ({
    chainId,
    addr: "0x" + tryNativeToHexString(address, "ethereum"),
  })) satisfies XAddressStruct[];

  const registerTasks = await Promise.allSettled(output.mockIntegrations.map(async ({chainId, address}) => {
    console.log(`Registering emitters for chainId ${chainId}`);
    const chain = getChain(chainId);

    // Loading this way would necessitate having last run enabled and we don't want that.
    // const mockIntegration = await getMockIntegration(chain);
    const signer = await getSigner(chain);
    const mockIntegration = MockRelayerIntegration__factory.connect(address, signer);

    const updateEmitters: typeof emitters = [];
    for (const emitter of emitters) {
      const currentEmitter = await mockIntegration.getRegisteredContract(emitter.chainId);
      if (currentEmitter.toLowerCase() !== emitter.addr.toLowerCase()) {
        updateEmitters.push(emitter);
      }
    }

    if (updateEmitters.length > 0) {
      const overrides = await buildOverrides(
        () => mockIntegration.estimateGas.registerEmitters(emitters),
        chain,
      );
      const receipt = await mockIntegration.registerEmitters(emitters, overrides).then(wait);

      if (receipt.status !== 1) {
        throw new Error(`Mock integration emitter registration failed for chain ${chainId}, tx id ${receipt.transactionHash}`);
      }
    }
  }));

  for (const task of registerTasks) {
    if (task.status === "rejected") {
      // These get discarded and need to be retried later with a separate invocation.
      console.log(task.reason?.stack || task.reason);
    }
  }
}

function isSupportedChain(deploy: Deployment): boolean {
  const item = operation.supportedChains.find((chain) => {
    return deploy.chainId === chain.chainId;
  });
  return item !== undefined;
}

run().then(() => console.log("Done!"));
