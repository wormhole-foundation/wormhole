import {
  init,
  ChainInfo,
  getDeliveryProvider,
  getOperatingChains,
} from "../helpers/env";
import { buildOverrides } from "../helpers/deployments";
import { wait } from "../helpers/utils";

const processName = "configureSupportedKeys";
init();
const operatingChains = getOperatingChains();


const keyTypes = {
  VAA: 1,
  CCTP: 2,
};

//TODO: configure which key types to update and whether to enable or disable them.
async function run() {
  console.log("Start! " + processName);

  const tasks = await Promise.allSettled(operatingChains.map((chain) => {
    return configureChainsDeliveryProvider(chain);
  }));

  for (const task of tasks) {
    if (task.status === "rejected") {
      console.log(`Failed to update supported message key types. ${task.reason?.stack || task.reason}`);
    }
  }
}

async function configureChainsDeliveryProvider(chain: ChainInfo) {
  console.log(
    "about to perform DeliveryProvider message key types update for chain " + chain.chainId
  );
  const deliveryProvider = await getDeliveryProvider(chain);

  for (const keyType of Object.values(keyTypes)) {
    const overrides = await buildOverrides(
      () => deliveryProvider.estimateGas.updateSupportedMessageKeyTypes(keyType, true),
      chain
    );
    await deliveryProvider
      .updateSupportedMessageKeyTypes(keyType, true, overrides)
      .then(wait);
  }

  console.log("done with DeliveryProvider message key types update on " + chain.chainId);
}

run().then(() => console.log("Done! " + processName));
