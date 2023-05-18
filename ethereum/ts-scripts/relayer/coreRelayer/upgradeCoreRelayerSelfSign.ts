import {
  deployCoreRelayerImplementation,
} from "../helpers/deployments";
import {
  init,
  ChainInfo,
  getCoreRelayer,
  writeOutputFiles,
  getOperatingChains,
} from "../helpers/env";
import {
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

  for (const chain of chains) {
    const coreRelayerImplementation = await deployCoreRelayerImplementation(
      chain,
    );
    await upgradeCoreRelayer(chain, coreRelayerImplementation.address);

    output.coreRelayerImplementations.push(coreRelayerImplementation);
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
