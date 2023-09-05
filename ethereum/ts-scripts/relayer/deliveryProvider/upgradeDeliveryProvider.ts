import {
  init,
  writeOutputFiles,
  ChainInfo,
  Deployment,
  getDeliveryProvider,
  getOperatingChains,
} from "../helpers/env";
import {
  buildOverrides,
  deployDeliveryProviderImplementation,
} from "../helpers/deployments";

const processName = "upgradeDeliveryProvider";
init();
const operatingChains = getOperatingChains();

interface DeliveryProviderUpgrade {
  deliveryProviderImplementations: Deployment[];
}

async function run() {
  console.log("Start!");
  const output: DeliveryProviderUpgrade = {
    deliveryProviderImplementations: [],
  };

  const tasks = await Promise.allSettled(
    operatingChains.map(async (chain) => {
      const implementation = await deployDeliveryProviderImplementation(chain);
      await upgradeDeliveryProvider(chain, implementation);

      return implementation;
    }),
  );
  for (const task of tasks) {
    if (task.status === "rejected") {
      console.log(
        `DeliveryProvider upgrade failed. ${task.reason?.stack || task.reason}`,
      );
    } else {
      output.deliveryProviderImplementations.push(task.value);
    }
  }

  writeOutputFiles(output, processName);
}

async function upgradeDeliveryProvider(
  operatingChain: ChainInfo,
  newImpl: Deployment,
) {
  console.log(
    "About to upgrade relay provider for chain " + operatingChain.chainId,
  );
  const provider = await getDeliveryProvider(operatingChain);

  const overrides = await buildOverrides(
    () => provider.estimateGas.upgrade(operatingChain.chainId, newImpl.address),
    operatingChain,
  );
  const tx = await provider.upgrade(
    operatingChain.chainId,
    newImpl.address,
    overrides,
  );
  const receipt = await tx.wait();

  if (receipt.status !== 1) {
    throw new Error(
      `Failed to upgrade DeliveryProvider on chain ${operatingChain.chainId}, tx id: ${tx.hash}`,
    );
  }
  console.log("Successfully upgraded relay provider " + operatingChain.chainId);
}

run().then(() => console.log("Done!"));
