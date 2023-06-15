import {
  deployDeliveryProviderImplementation,
  deployDeliveryProviderProxy,
  deployDeliveryProviderSetup,
} from "../helpers/deployments";
import { Deployment, getOperatingChains, init, writeOutputFiles } from "../helpers/env";

const processName = "deployDeliveryProvider";
init();
const chains = getOperatingChains();

async function run() {
  console.log(`Start ${processName}!`);
  const output: Record<string, Deployment[]> = {
    deliveryProviderImplementations: [],
    deliveryProviderSetups: [],
    deliveryProviderProxies: [],
  };

  for (const chain of chains) {
    console.log(`Deploying for chain ${chain.chainId}...`);
    const deliveryProviderImplementation = await deployDeliveryProviderImplementation(
      chain
    );
    const deliveryProviderSetup = await deployDeliveryProviderSetup(chain);
    const deliveryProviderProxy = await deployDeliveryProviderProxy(
      chain,
      deliveryProviderSetup.address,
      deliveryProviderImplementation.address
    );
    output.deliveryProviderImplementations.push(deliveryProviderImplementation);
    output.deliveryProviderSetups.push(deliveryProviderSetup);
    output.deliveryProviderProxies.push(deliveryProviderProxy);
    console.log("");
  }

  writeOutputFiles(output, processName);
}

run().then(() => console.log("Done!"));
