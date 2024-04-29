import {
  buildOverrides,
  deployWormholeRelayerImplementation,
} from "../helpers/deployments";
import {
  init,
  ChainInfo,
  getWormholeRelayer,
  saveDeployments,
  getOperatingChains,
  Deployment,
} from "../helpers/env";
import { createWormholeRelayerUpgradeVAA } from "../helpers/vaa";

const processName = "upgradeWormholeRelayerSelfSign";
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
      await upgradeWormholeRelayer(chain, implementation.address);

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

  saveDeployments(output, processName);
}

async function upgradeWormholeRelayer(
  chain: ChainInfo,
  newImplementationAddress: string,
) {
  console.log("upgradeWormholeRelayer " + chain.chainId);

  const wormholeRelayer = await getWormholeRelayer(chain);

  const vaa = createWormholeRelayerUpgradeVAA(chain, newImplementationAddress);

  const overrides = await buildOverrides(
    () => wormholeRelayer.estimateGas.submitContractUpgrade(vaa),
    chain,
  );
  const tx = await wormholeRelayer.submitContractUpgrade(vaa, overrides);

  const receipt = await tx.wait();

  if (receipt.status !== 1) {
    throw new Error(
      `Failed to upgrade on chain ${chain.chainId}, tx id: ${tx.hash}`,
    );
  }
  console.log(
    "Successfully upgraded the core relayer contract on " + chain.chainId,
  );
}

run().then(() => console.log("Done! " + processName));
