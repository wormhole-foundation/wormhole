import { tryNativeToHexString } from "@certusone/wormhole-sdk";
import {
  deployCoreRelayerImplementation,
  deployCoreRelayerLibrary,
} from "../helpers/deployments";
import {
  init,
  loadChains,
  ChainInfo,
  getCoreRelayer,
  getRelayProviderAddress,
  getCoreRelayerAddress,
  writeOutputFiles,
  getOperatingChains,
} from "../helpers/env";
import {
  createRegisterChainVAA,
  createDefaultRelayProviderVAA,
  createCoreRelayerUpgradeVAA,
} from "../helpers/vaa";

const processName = "upgradeCoreRelayerSelfSign";
init();
const chains = getOperatingChains();

async function run() {
  console.log("Start!");
  const output: any = {
    coreRelayerImplementations: [],
    coreRelayerLibraries: [],
  };

  for (let i = 0; i < chains.length; i++) {
    const coreRelayerLibrary = await deployCoreRelayerLibrary(chains[i]);
    const coreRelayerImplementation = await deployCoreRelayerImplementation(
      chains[i],
      coreRelayerLibrary.address
    );
    await upgradeCoreRelayer(chains[i], coreRelayerImplementation.address);

    output.coreRelayerImplementations.push(coreRelayerImplementation);
    output.coreRelayerLibraries.push(coreRelayerLibrary);
  }

  writeOutputFiles(output, processName);
}

async function upgradeCoreRelayer(
  chain: ChainInfo,
  newImplementationAddress: string
) {
  console.log("upgradeCoreRelayer " + chain.chainId);

  const coreRelayer = await getCoreRelayer(chain);

  await coreRelayer.submitContractUpgrade(
    createCoreRelayerUpgradeVAA(chain, newImplementationAddress)
  );

  console.log(
    "Successfully upgraded the core relayer contract on " + chain.chainId
  );
}

run().then(() => console.log("Done! " + processName));
