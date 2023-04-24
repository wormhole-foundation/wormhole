import {
  deployCoreRelayerImplementation,
  deployCoreRelayerProxy,
  deployCoreRelayerSetup,
  deployForwardWrapper,
} from "../helpers/deployments";
import {
  init,
  writeOutputFiles,
  getRelayProviderAddress,
  getOperatingChains,
  getCoreRelayerAddress,
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
    const forwardWrapper = await deployForwardWrapper(
      chain,
      // uses create2 to determine address before deployment
      await getCoreRelayerAddress(chain)
    );
    const coreRelayerImplementation = await deployCoreRelayerImplementation(
      chain,
      forwardWrapper.address
    );
    const coreRelayerSetup = await deployCoreRelayerSetup(chain);
    const coreRelayerProxy = await deployCoreRelayerProxy(
      chain,
      coreRelayerSetup.address,
      coreRelayerImplementation.address,
      chain.wormholeAddress,
      getRelayProviderAddress(chain)
    );

    output.coreRelayerLibraries.push(forwardWrapper);
    output.coreRelayerImplementations.push(coreRelayerImplementation);
    output.coreRelayerSetups.push(coreRelayerSetup);
    output.coreRelayerProxies.push(coreRelayerProxy);
    console.log("");
  }

  writeOutputFiles(output, processName);
}

run().then(() => console.log("Done! " + processName));
