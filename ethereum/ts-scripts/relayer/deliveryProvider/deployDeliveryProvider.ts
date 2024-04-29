import { inspect } from "util";
import {
  deployDeliveryProviderImplementation,
  deployDeliveryProviderProxy,
  deployDeliveryProviderSetup,
} from "../helpers/deployments";
import {
  Deployment,
  getOperationDescriptor,
  init,
  saveDeployments,
} from "../helpers/env";

const processName = "deployDeliveryProvider";
init();
const operation = getOperationDescriptor();

interface DeliveryProviderDeployment {
  deliveryProviderImplementations: Deployment[];
  deliveryProviderSetups: Deployment[];
  deliveryProviders: Deployment[];
}

async function run() {
  console.log(`Start ${processName}!`);

  const deployments: DeliveryProviderDeployment = {
    deliveryProviderImplementations: [],
    deliveryProviderSetups: [],
    deliveryProviders: [],
  };

  const tasks = await Promise.allSettled(
    operation.operatingChains.map(async (chain) => {
      console.log(`Deploying for chain ${chain.chainId}...`);
      const deliveryProviderImplementation =
        await deployDeliveryProviderImplementation(chain);
      const deliveryProviderSetup = await deployDeliveryProviderSetup(chain);
      const deliveryProvider = await deployDeliveryProviderProxy(
        chain,
        deliveryProviderSetup.address,
        deliveryProviderImplementation.address,
      );

      return {
        deliveryProviderImplementation,
        deliveryProviderSetup,
        deliveryProvider,
      };
    }),
  );

  for (const task of tasks) {
    if (task.status === "rejected") {
      // TODO: add chain as context
      // These get discarded and need to be retried later with a separate invocation.
      console.log(
        `Deployment failed: ${task.reason?.stack || inspect(task.reason)}`,
      );
    } else {
      deployments.deliveryProviderImplementations.push(
        task.value.deliveryProviderImplementation,
      );
      deployments.deliveryProviderSetups.push(task.value.deliveryProviderSetup);
      deployments.deliveryProviders.push(task.value.deliveryProvider);
    }
  }

  saveDeployments(deployments, processName);
}

run().then(() => console.log("Done!"));
