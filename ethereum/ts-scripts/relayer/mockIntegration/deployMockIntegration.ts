import { inspect } from "util";
import {
  init,
  saveDeployments,
  Deployment,
  getOperationDescriptor,
  loadMockIntegrations,
  getChain,
} from "../helpers/env";
import { deployMockIntegration } from "../helpers/deployments";
import { XAddressStruct } from "../../../ethers-contracts/MockRelayerIntegration";
import { printRegistration, registerMockIntegration } from "./mockIntegrationDeploy";
import { nativeEvmAddressToHex } from "../helpers/utils";

const processName = "deployMockIntegration";
init();
const operation = getOperationDescriptor();

interface MockIntegrationDeployment {
  mockIntegrations: Deployment[];
}

async function run() {
  console.log("Start!");

  const newDeployments: Deployment[] = [];

  // TODO: deploy only on chains missing deployment
  const deploymentTasks = await Promise.allSettled(
    operation.operatingChains.map(async (chain) => {
      return deployMockIntegration(chain);
    }),
  );

  let failed = false;
  for (const task of deploymentTasks) {
    if (task.status === "rejected") {
      // TODO: add chain as context
      // These get discarded and need to be retried later with a separate invocation.
      console.log(
        `Deployment failed: ${task.reason?.stack || inspect(task.reason)}`,
      );
      failed = true;
    } else {
      newDeployments.push(task.value);
    }
  }

  const output = {
    mockIntegrations: newDeployments,
  } satisfies MockIntegrationDeployment;
  saveDeployments(output, processName);

  const mockIntegrations = loadMockIntegrations();
  const emitters = loadMockIntegrations().map(({ address, chainId }) => ({
    chainId,
    addr: nativeEvmAddressToHex(address)
  })) satisfies XAddressStruct[];

  const registerTasks = await Promise.allSettled(
    mockIntegrations.map(async ({ chainId }) => {
      const chain = getChain(chainId);
      return registerMockIntegration(chain, emitters);
    }),
  );

  for (const task of registerTasks) {
    if (task.status === "rejected") {
      // These get discarded and need to be retried later with a separate invocation.
      console.log(task.reason?.stack || inspect(task.reason));
      failed = true;
    } else {
      printRegistration(task.value.updateEmitters, task.value.chain);
    }
  }

  // We throw here to ensure non zero exit code and communicate failure to shell
  if (failed) {
    throw new Error("One or more errors happened during execution. See messages above.");
  }
}

run().then(() => console.log("Done!"));
