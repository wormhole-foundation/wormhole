import { tryNativeToHexString } from "@certusone/wormhole-sdk";

import {
  init,
  loadChains,
  ChainInfo,
  getCoreRelayerAddress,
  getRelayProvider,
  getRelayProviderAddress,
  getOperatingChains,
} from "../helpers/env";
import { wait } from "../helpers/utils";

const processName = "registerChainsRelayProvider";
init();
const operatingChains = getOperatingChains();
const chains = loadChains();

async function run() {
  console.log("Start! " + processName);

  for (let i = 0; i < operatingChains.length; i++) {
    await registerChainsRelayProvider(operatingChains[i]);
  }
}

async function registerChainsRelayProvider(chain: ChainInfo) {
  console.log("about to perform registrations for chain " + chain.chainId);

  const relayProvider = getRelayProvider(chain);
  const coreRelayerAddress = getCoreRelayerAddress(chain);

  await relayProvider.updateCoreRelayer(coreRelayerAddress).then(wait);

  for (let i = 0; i < chains.length; i++) {
    console.log(`Cross registering with chain ${chains[i].chainId}...`);
    const targetChainProviderAddress = getRelayProviderAddress(chains[i]);
    const whAddress =
      "0x" + tryNativeToHexString(targetChainProviderAddress, "ethereum");

    await relayProvider.updateSupportedChain(chains[i].chainId, true);
    await relayProvider.updateTargetChainAddress(whAddress, chains[i].chainId);
  }

  console.log("done with registrations on " + chain.chainId);
}

run().then(() => console.log("Done! " + processName));
