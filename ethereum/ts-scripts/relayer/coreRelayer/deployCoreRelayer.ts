import {
  deployCoreRelayerImplementation,
  deployCoreRelayerLibrary,
  deployCoreRelayerProxy,
  deployCoreRelayerSetup,
} from "../helpers/deployments";
import {
  init,
  loadChains,
  writeOutputFiles,
  getRelayProviderAddress,
  getOperatingChains,
} from "../helpers/env";

const processName = "deployCoreRelayer";
init();
const chains = getOperatingChains();

async function run() {
  console.log("Start! " + processName);

  const output: any = {
    coreRelayerLibraries: [],
    coreRelayerImplementations: [],
    coreRelayerSetups: [],
    coreRelayerProxies: [],
  };

  for (const chain of chains) {
    console.log(`Deploying for chain ${chain.chainId}...`);
    const coreRelayerLibrary = await deployCoreRelayerLibrary(chain);
    const coreRelayerImplementation = await deployCoreRelayerImplementation(
      chain,
      coreRelayerLibrary.address
    );
    const coreRelayerSetup = await deployCoreRelayerSetup(chain);
    const coreRelayerProxy = await deployCoreRelayerProxy(
      chain,
      coreRelayerSetup.address,
      coreRelayerImplementation.address,
      chain.wormholeAddress,
      getRelayProviderAddress(chain)
    );

    output.coreRelayerLibraries.push(coreRelayerLibrary);
    output.coreRelayerImplementations.push(coreRelayerImplementation);
    output.coreRelayerSetups.push(coreRelayerSetup);
    output.coreRelayerProxies.push(coreRelayerProxy);
    console.log("");
  }

  writeOutputFiles(output, processName);
}

run().then(() => console.log("Done! " + processName));
