import {
  init,
  writeOutputFiles,
  ChainInfo,
  Deployment,
  getRelayProvider,
  getOperatingChains,
} from "../helpers/env";
import { deployRelayProviderImplementation } from "../helpers/deployments";

const processName = "upgradeRelayProvider";
init();
const chains = getOperatingChains();

async function run() {
  console.log("Start!");
  const output: any = {
    relayProviderImplementations: [],
  };

  for (let i = 0; i < chains.length; i++) {
    const relayProviderImplementation = await deployRelayProviderImplementation(
      chains[i]
    );
    await upgradeRelayProvider(chains[i], relayProviderImplementation);

    output.relayProviderImplementations.push(relayProviderImplementation);
  }

  writeOutputFiles(output, processName);
}

async function upgradeRelayProvider(chain: ChainInfo, newImpl: Deployment) {
  console.log("About to upgrade relay provider for chain " + chain.chainId);
  const provider = getRelayProvider(chain);
  await provider.upgrade(chain.chainId, newImpl.address);
  console.log("Successfully upgraded relay provider " + chain.chainId);
}

run().then(() => console.log("Done!"));
