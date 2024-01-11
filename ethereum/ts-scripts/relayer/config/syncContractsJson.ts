import { readFileSync, writeFileSync } from "fs";
import {
  getWormholeRelayer,
  getCreate2Factory,
  getMockIntegration,
  getDeliveryProvider,
  init,
  loadChains,
  updateContractAddress,
  Deployment,
} from "../helpers/env";

const env = init();
const chains = loadChains();

interface ContractsJson {
  deliveryProviders: Deployment[];
  wormholeRelayers: Deployment[];
  mockIntegrations: Deployment[];
  create2Factories: Deployment[];
}

async function main() {
  const path = `./ts-scripts/relayer/config/${env}/contracts.json`;
  const contractsFile = readFileSync(path, "utf8");
  const contracts: ContractsJson = JSON.parse(contractsFile);
  console.log(`Old:\n${contractsFile}`);
  contracts.deliveryProviders = [];
  contracts.wormholeRelayers = [];
  contracts.mockIntegrations = [];
  contracts.create2Factories = [];
  for (const chain of chains) {
    updateContractAddress(contracts.deliveryProviders, {
      chainId: chain.chainId,
      address: (await getDeliveryProvider(chain)).address,
    });
    updateContractAddress(contracts.wormholeRelayers, {
      chainId: chain.chainId,
      address: (await getWormholeRelayer(chain)).address,
    });
    updateContractAddress(contracts.mockIntegrations, {
      chainId: chain.chainId,
      address: (await getMockIntegration(chain)).address,
    });
    updateContractAddress(contracts.create2Factories, {
      chainId: chain.chainId,
      address: (await getCreate2Factory(chain)).address,
    });
  }
  const newStr = JSON.stringify(contracts, undefined, 2);
  console.log(`New:\n${newStr}`);
  writeFileSync(path, newStr);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
