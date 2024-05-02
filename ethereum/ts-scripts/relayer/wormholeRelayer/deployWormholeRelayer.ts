import { inspect } from "util";
import {
  deployWormholeRelayerImplementation,
  deployWormholeRelayerProxy,
} from "../helpers/deployments";
import {
  init,
  saveDeployments,
  getDeliveryProviderAddress,
  Deployment,
  getOperationDescriptor,
} from "../helpers/env";

const processName = "deployWormholeRelayer";
init();
const operation = getOperationDescriptor();

interface WormholeRelayerDeployment {
  wormholeRelayerImplementations: Deployment[];
  wormholeRelayers: Deployment[];
}

async function run() {
  console.log("Start! " + processName);

  const deployments: WormholeRelayerDeployment = {
    wormholeRelayerImplementations: [],
    wormholeRelayers: [],
  };

  const tasks = await Promise.allSettled(
    operation.operatingChains.map(async (chain) => {
      console.log(`Deploying for chain ${chain.chainId}...`);
      const wormholeRelayerImplementation =
        await deployWormholeRelayerImplementation(chain);
      const wormholeRelayer = await deployWormholeRelayerProxy(
        chain,
        wormholeRelayerImplementation.address,
        getDeliveryProviderAddress(chain),
      );

      return {
        wormholeRelayerImplementation,
        wormholeRelayer,
      };
    }),
  );

  let failed = false;
  for (const task of tasks) {
    if (task.status === "rejected") {
      // TODO: add chain as context
      // These get discarded and need to be retried later with a separate invocation.
      console.log(
        `Deployment failed: ${task.reason?.stack || inspect(task.reason)}`,
      );
      failed = true;
    } else {
      deployments.wormholeRelayerImplementations.push(
        task.value.wormholeRelayerImplementation,
      );
      deployments.wormholeRelayers.push(task.value.wormholeRelayer);
    }
  }

  saveDeployments(deployments, processName);

  // We throw here to ensure non zero exit code and communicate failure to shell
  if (failed) {
    throw new Error("One or more errors happened during execution. See messages above.");
  }
}

run().then(() => console.log("Done! " + processName));
