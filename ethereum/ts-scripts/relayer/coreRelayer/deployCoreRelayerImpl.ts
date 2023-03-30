import {
  deployCoreRelayerImplementation,
  deployCoreRelayerLibrary,
} from "../helpers/deployments";
import {
  getOperatingChains,
  init,
  loadChains,
  writeOutputFiles,
} from "../helpers/env";

const processName = "deployCoreRelayerImpl";
init();
const chains = getOperatingChains();

async function run() {
  console.log("Start! " + processName);

  const output: any = {
    coreRelayerLibraries: [],
    coreRelayerImplementations: [],
  };

  for (let i = 0; i < chains.length; i++) {
    const coreRelayerLibrary = await deployCoreRelayerLibrary(chains[i]);
    const coreRelayerImplementation = await deployCoreRelayerImplementation(
      chains[i],
      coreRelayerLibrary.address
    );
    output.coreRelayerImplementations.push(coreRelayerImplementation);
    output.coreRelayerLibraries.push(coreRelayerLibrary);
  }

  writeOutputFiles(output, processName);
}

run().then(() => console.log("Done! " + processName));
