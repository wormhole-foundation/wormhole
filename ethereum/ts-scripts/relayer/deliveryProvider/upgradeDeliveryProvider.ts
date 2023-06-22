import {
  init,
  writeOutputFiles,
  ChainInfo,
  Deployment,
  getDeliveryProvider,
  getOperatingChains,
} from "../helpers/env";
import { deployDeliveryProviderImplementation } from "../helpers/deployments";

const processName = "upgradeDeliveryProvider";
init();
const chains = getOperatingChains();

async function run() {
  console.log("Start!");
  const output: any = {
    deliveryProviderImplementations: [],
  };

  for (let i = 0; i < chains.length; i++) {
    const deliveryProviderImplementation = await deployDeliveryProviderImplementation(
      chains[i]
    );
    await upgradeDeliveryProvider(chains[i], deliveryProviderImplementation);

    output.deliveryProviderImplementations.push(deliveryProviderImplementation);
  }

  writeOutputFiles(output, processName);
}

async function upgradeDeliveryProvider(chain: ChainInfo, newImpl: Deployment) {
  console.log("About to upgrade relay provider for chain " + chain.chainId);
  const provider = getDeliveryProvider(chain);
  const tx = await provider.upgrade(chain.chainId, newImpl.address);
  await tx.wait();
  console.log("Successfully upgraded relay provider " + chain.chainId);
}

run().then(() => console.log("Done!"));
