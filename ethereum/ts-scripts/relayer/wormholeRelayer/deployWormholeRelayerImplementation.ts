import {
  deployWormholeRelayerImplementation,
} from "../helpers/deployments";
import {
  init,
  writeOutputFiles,
  getOperatingChains,
  Deployment,
} from "../helpers/env";

const processName = "deployWormholeRelayerImplementation";
init();
const operatingChains = getOperatingChains();

interface WormholeRelayerUpgrade {
  wormholeRelayerImplementations: Deployment[];
}

async function run() {
  console.log("Start!");
  const output: WormholeRelayerUpgrade = {
    wormholeRelayerImplementations: [],
    
  };

  const tasks = await Promise.allSettled(
    operatingChains.map(async (chain) => {
      const implementation = await deployWormholeRelayerImplementation(chain);
      return implementation;
    }),
  );

  for (const task of tasks) {
    if (task.status === "rejected") {
      console.log(`WormholeRelayer upgrade failed. ${task.reason?.stack || task.reason}`);
    } else {
      output.wormholeRelayerImplementations.push(task.value);
    }
  }

  writeOutputFiles(output, processName);
}

run().then(() => console.log("Done! " + processName));
