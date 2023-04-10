import {
  deployRelayProviderImplementation,
  deployRelayProviderProxy,
  deployRelayProviderSetup,
} from "../helpers/deployments";
import {
  getOperatingChains,
  getSigner,
  init,
  loadChains,
  loadPrivateKey,
  writeOutputFiles,
} from "../helpers/env";

const processName = "deployRelayProvider";
init();
const chains = getOperatingChains();
const privateKey = loadPrivateKey();

async function run() {
  console.log(`Start ${processName}!`);
  const output: any = {
    relayProviderImplementations: [],
    relayProviderSetups: [],
    relayProviderProxies: [],
  };

  for (const chain of chains) {
    console.log(`Deploying for chain ${chain.chainId}...`);
    const relayProviderImplementation = await deployRelayProviderImplementation(
      chain
    );
    const relayProviderSetup = await deployRelayProviderSetup(chain);
    const relayProviderProxy = await deployRelayProviderProxy(
      chain,
      relayProviderSetup.address,
      relayProviderImplementation.address
    );

    output.relayProviderImplementations.push(relayProviderImplementation);
    output.relayProviderSetups.push(relayProviderSetup);
    output.relayProviderProxies.push(relayProviderProxy);
    console.log("");
  }

  writeOutputFiles(output, processName);
}

run().then(() => console.log("Done!"));
