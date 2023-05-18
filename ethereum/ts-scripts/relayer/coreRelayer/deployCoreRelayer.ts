import {
  deployCoreRelayerImplementation,
  deployCoreRelayerProxy,
} from "../helpers/deployments";
import {
  init,
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
    coreRelayerImplementations: [],
    coreRelayerProxies: [],
  };

  for (const chain of chains) {
    console.log(`Deploying for chain ${chain.chainId}...`);
    const coreRelayerImplementation = await deployCoreRelayerImplementation(
      chain,
    );
    const coreRelayerProxy = await deployCoreRelayerProxy(
      chain,
      coreRelayerImplementation.address,
      getRelayProviderAddress(chain),
    );

    output.coreRelayerImplementations.push(coreRelayerImplementation);
    output.coreRelayerProxies.push(coreRelayerProxy);
    console.log("");
  }

  writeOutputFiles(output, processName);
}

run().then(() => console.log("Done! " + processName));
