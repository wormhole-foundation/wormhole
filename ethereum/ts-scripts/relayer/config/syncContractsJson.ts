import { readFileSync, writeFileSync } from "fs";
import {
  init,
  updateContractAddress,
  Deployment,
  loadDeliveryProviders,
  loadWormholeRelayers,
  loadMockIntegrations,
  loadCreate2Factories,
  loadWormholeRelayerImplementations,
} from "../helpers/env";

const env = init();

interface ContractsJson {
  deliveryProviders: Deployment[];
  wormholeRelayers: Deployment[];
  mockIntegrations: Deployment[];
  create2Factories: Deployment[];
  wormholeRelayerImplementations: Deployment[];
}

async function main() {
  const path = `./ts-scripts/relayer/config/${env}/contracts.json`;
  const contractsFile = readFileSync(path, "utf8");
  const contracts: ContractsJson = JSON.parse(contractsFile);
  console.log(`Old:\n${contractsFile}`);
  contracts.create2Factories = loadCreate2Factories();
  contracts.wormholeRelayers = loadWormholeRelayers();
  contracts.wormholeRelayerImplementations = loadWormholeRelayerImplementations();
  contracts.deliveryProviders = loadDeliveryProviders();
  contracts.mockIntegrations = loadMockIntegrations();

  const newStr = JSON.stringify(contracts, undefined, 2);
  console.log(`New:\n${newStr}`);
  writeFileSync(path, newStr);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
