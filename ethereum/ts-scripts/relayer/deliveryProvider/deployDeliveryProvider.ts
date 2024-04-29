import {
  deployDeliveryProviderImplementation,
  deployDeliveryProviderProxy,
  deployDeliveryProviderSetup,
} from "../helpers/deployments";
import {
  Deployment,
  getOperationDescriptor,
  init,
  writeOutputFiles,
  loadDeliveryProviderImplementations,
  loadDeliveryProviderSetups,
  loadDeliveryProviders,
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
    deliveryProviderImplementations: loadDeliveryProviderImplementations().filter(isSupportedChain),
    deliveryProviderSetups: loadDeliveryProviderSetups().filter(isSupportedChain),
    deliveryProviders: loadDeliveryProviders().filter(isSupportedChain),
  };

  for (const chain of operation.operatingChains) {
    console.log(`Deploying for chain ${chain.chainId}...`);
    const deliveryProviderImplementation =
      await deployDeliveryProviderImplementation(chain);
    const deliveryProviderSetup = await deployDeliveryProviderSetup(chain);
    const deliveryProviderProxy = await deployDeliveryProviderProxy(
      chain,
      deliveryProviderSetup.address,
      deliveryProviderImplementation.address,
    );

    deployments.deliveryProviderImplementations.push(
      deliveryProviderImplementation,
    );
    deployments.deliveryProviderSetups.push(deliveryProviderSetup);
    deployments.deliveryProviders.push(deliveryProviderProxy);
    console.log("");
  }

  writeOutputFiles(deployments, processName);
}

function isSupportedChain(deploy: Deployment): boolean {
  const item = operation.supportedChains.find((chain) => {
    return deploy.chainId === chain.chainId;
  });
  return item !== undefined;
}

run().then(() => console.log("Done!"));
