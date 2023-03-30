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

  for (let i = 0; i < chains.length; i++) {
    console.log(`Deploying for chain ${chains[i].chainId}...`);
    const coreRelayerLibrary = await deployCoreRelayerLibrary(chains[i]);
    const coreRelayerImplementation = await deployCoreRelayerImplementation(
      chains[i],
      coreRelayerLibrary.address
    );
    const coreRelayerSetup = await deployCoreRelayerSetup(chains[i]);
    const coreRelayerProxy = await deployCoreRelayerProxy(
      chains[i],
      coreRelayerSetup.address,
      coreRelayerImplementation.address,
      chains[i].wormholeAddress,
      getRelayProviderAddress(chains[i])
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
