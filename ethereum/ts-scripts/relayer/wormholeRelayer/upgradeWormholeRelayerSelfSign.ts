import { deployWormholeRelayerImplementation } from "../helpers/deployments";
import {
  init,
  ChainInfo,
  getWormholeRelayer,
  writeOutputFiles,
  getOperatingChains,
} from "../helpers/env";
import { createWormholeRelayerUpgradeVAA } from "../helpers/vaa";

const processName = "upgradeWormholeRelayerSelfSign";
init();
const chains = getOperatingChains();

async function run() {
  console.log("Start!");
  const output: any = {
    wormholeRelayerImplementations: []
  };

  for (const chain of chains) {
    const coreRelayerImplementation = await deployWormholeRelayerImplementation(
      chain
    );
    await upgradeWormholeRelayer(chain, coreRelayerImplementation.address);

    output.wormholeRelayerImplementations.push(coreRelayerImplementation);
  }

  writeOutputFiles(output, processName);
}

async function upgradeWormholeRelayer(
  chain: ChainInfo,
  newImplementationAddress: string
) {
  console.log("upgradeWormholeRelayer " + chain.chainId);

  const coreRelayer = await getWormholeRelayer(chain);

  const tx = await coreRelayer.submitContractUpgrade(
    createWormholeRelayerUpgradeVAA(chain, newImplementationAddress)
  );

  await tx.wait();

  console.log(
    "Successfully upgraded the core relayer contract on " + chain.chainId
  );
}

run().then(() => console.log("Done! " + processName));
