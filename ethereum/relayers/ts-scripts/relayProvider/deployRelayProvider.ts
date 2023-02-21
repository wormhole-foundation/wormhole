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

  for (let i = 0; i < chains.length; i++) {
    console.log(`Deploying for chain ${chains[i].chainId}...`);
    const relayProviderImplementation = await deployRelayProviderImplementation(
      chains[i]
    );
    const relayProviderSetup = await deployRelayProviderSetup(chains[i]);
    const relayProviderProxy = await deployRelayProviderProxy(
      chains[i],
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
