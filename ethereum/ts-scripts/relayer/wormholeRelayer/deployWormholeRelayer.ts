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
  loadWormholeRelayerImplementations,
  loadWormholeRelayers,
} from "../helpers/env";

const processName = "deployWormholeRelayer";
init();
const operation = getOperationDescriptor();

interface WormholeRelayerDeployment {
  wormholeRelayerImplementations: Deployment[];
  wormholeRelayers: Deployment[];
}

async function run() {
  console.log("Start! " + processName);

  const deployments: WormholeRelayerDeployment = {
    wormholeRelayerImplementations: loadWormholeRelayerImplementations().filter(isSupportedChain) || [],
    wormholeRelayers: loadWormholeRelayers(false).filter(isSupportedChain) || [],
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
    deployments.wormholeRelayers.push(coreRelayerProxy);
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
