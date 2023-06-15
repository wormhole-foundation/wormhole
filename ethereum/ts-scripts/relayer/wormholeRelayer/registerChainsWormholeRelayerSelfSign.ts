import {
  init,
  loadChains,
  ChainInfo,
  getWormholeRelayer,
  getOperatingChains,
} from "../helpers/env";
import { wait } from "../helpers/utils";
import { createRegisterChainVAA } from "../helpers/vaa";

const processName = "registerChainsWormholeRelayerSelfSign";
init();
const operatingChains = getOperatingChains();
const chains = loadChains();

async function run() {
  console.log("Start! " + processName);

  for (const operatingChain of operatingChains) {
    await registerChainsWormholeRelayer(operatingChain);
  }
}

async function registerChainsWormholeRelayer(chain: ChainInfo) {
  console.log("registerChainsWormholeRelayer " + chain.chainId);

  const coreRelayer = await getWormholeRelayer(chain);
  for (const targetChain of chains) {
    await coreRelayer
      .registerWormholeRelayerContract(createRegisterChainVAA(targetChain))
      .then(wait);
  }

  console.log(
    "Did all contract registrations for the core relayer on " + chain.chainId
  );
}

run().then(() => console.log("Done! " + processName));
