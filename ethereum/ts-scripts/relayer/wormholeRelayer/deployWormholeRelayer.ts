import {
  deployWormholeRelayerImplementation,
  deployWormholeRelayerProxy,
} from "../helpers/deployments";
import {
  init,
  writeOutputFiles,
  getDeliveryProviderAddress,
  Deployment,
  getOperationDescriptor,
  loadLastRun,
} from "../helpers/env";

const processName = "deployWormholeRelayer";
init();
const operation = getOperationDescriptor();

interface WormholeRelayerDeployment {
  wormholeRelayerImplementations: Deployment[];
  wormholeRelayerProxies: Deployment[];
}

async function run() {
  console.log("Start! " + processName);

  const lastRun: WormholeRelayerDeployment | undefined =
    loadLastRun(processName);
  const deployments: WormholeRelayerDeployment = {
    wormholeRelayerImplementations: lastRun?.wormholeRelayerImplementations?.filter(isSupportedChain) || [],
    wormholeRelayerProxies: lastRun?.wormholeRelayerProxies?.filter(isSupportedChain) || [],
  };

  for (const chain of operation.operatingChains) {
    console.log(`Deploying for chain ${chain.chainId}...`);
    const relayerImplementation = await deployWormholeRelayerImplementation(
      chain,
    );
    const coreRelayerProxy = await deployWormholeRelayerProxy(
      chain,
      relayerImplementation.address,
      getDeliveryProviderAddress(chain),
    );

    deployments.wormholeRelayerImplementations.push(relayerImplementation);
    deployments.wormholeRelayerProxies.push(coreRelayerProxy);
  }

  writeOutputFiles(deployments, processName);
}

function isSupportedChain(deploy: Deployment): boolean {
  const item = operation.supportedChains.find((chain) => {
    return deploy.chainId === chain.chainId;
  });
  return item !== undefined;
}

run().then(() => console.log("Done! " + processName));
