import {
  buildOverrides,
  deployWormholeRelayerImplementation,
} from "../helpers/deployments";
import {
  init,
  ChainInfo,
  getWormholeRelayer,
  writeOutputFiles,
  getOperatingChains,
  GovernanceDeployment,
} from "../helpers/env";
import { createWormholeRelayerUpgradeVAA } from "../helpers/vaa";

const processName = "upgradeWormholeRelayerSelfSign";
init();
const operatingChains = getOperatingChains();



interface WormholeRelayerUpgrade {
  wormholeRelayerImplementations: GovernanceDeployment[];
}

async function run() {
  console.log("Start!");
  const output: WormholeRelayerUpgrade = {
    wormholeRelayerImplementations: [],
    
  };

  const tasks = await Promise.allSettled(
    operatingChains.map(async (chain) => {
      const implementation = await deployWormholeRelayerImplementation(chain);
      const vaa = createWormholeRelayerUpgradeVAA(chain, implementation.address);

      console.log(`Upgrade wormhole relayer implementation on ${chain.chainId}:\n${vaa}`);

      return { ...implementation, vaa };
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
