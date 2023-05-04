import {
  init,
  loadChains,
  ChainInfo,
  getCoreRelayer,
  getOperatingChains,
} from "../helpers/env";
import { wait } from "../helpers/utils";
import {
  createRegisterChainVAA,
} from "../helpers/vaa";

const processName = "registerChainsCoreRelayerSelfSign";
init();
const operatingChains = getOperatingChains();
const chains = loadChains();

async function run() {
  console.log("Start! " + processName);

  for (const operatingChain of operatingChains) {
    await registerChainsCoreRelayer(operatingChain);
  }
}

async function registerChainsCoreRelayer(chain: ChainInfo) {
  console.log("registerChainsCoreRelayer " + chain.chainId);

  const coreRelayer = await getCoreRelayer(chain);
  for (const targetChain of chains) {
    await coreRelayer
      .registerCoreRelayerContract(createRegisterChainVAA(targetChain))
      .then(wait);
  }

  console.log(
    "Did all contract registrations for the core relayer on " + chain.chainId
  );
}

run().then(() => console.log("Done! " + processName));
