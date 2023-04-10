import type { ChainId } from "@certusone/wormhole-sdk";
import { ethers, Signer } from "ethers";
import fs from "fs";

import { CoreRelayer } from "../../../ethers-contracts/CoreRelayer";
import { RelayProvider } from "../../../ethers-contracts/RelayProvider";
import { MockRelayerIntegration } from "../../../ethers-contracts/MockRelayerIntegration";

import { RelayProvider__factory } from "../../../ethers-contracts/factories/RelayProvider__factory";
import { CoreRelayer__factory } from "../../../ethers-contracts/factories/CoreRelayer__factory";
import { MockRelayerIntegration__factory } from "../../../ethers-contracts/factories/MockRelayerIntegration__factory";

export type ChainInfo = {
  evmNetworkId: number;
  chainId: ChainId;
  rpc: string;
  wormholeAddress: string;
};

export type Deployment = {
  chainId: ChainId;
  address: string;
};

const DEFAULT_ENV = "testnet";

export let env = "";
let lastRunOverride: boolean | undefined;

export function init(overrides: { lastRunOverride?: boolean } = {}): string {
  env = get_env_var("ENV");
  if (!env) {
    console.log(
      "No environment was specified, using default environment files"
    );
    env = DEFAULT_ENV;
  }
  lastRunOverride = overrides?.lastRunOverride;

  require("dotenv").config({
    path: `./ts-scripts/relayer/.env${env != DEFAULT_ENV ? "." + env : ""}`,
  });
  return env;
}

function get_env_var(env: string): string {
  const v = process.env[env];
  return v || "";
}

function getContainer(): string | null {
  const container = get_env_var("CONTAINER");
  if (!container) {
    return null;
  }

  return container;
}

export function loadScriptConfig(processName: string): any {
  const configFile = fs.readFileSync(
    `./ts-scripts/relayer/config/${env}/scriptConfigs/${processName}.json`
  );
  const config = JSON.parse(configFile.toString());
  if (!config) {
    throw Error("Failed to pull config file!");
  }
  return config;
}

export function getOperatingChains(): ChainInfo[] {
  const allChains = loadChains();
  const container = getContainer();
  let operatingChains = null;

  if (container == "evm1") {
    operatingChains = [2];
  }
  if (container == "evm2") {
    operatingChains = [4];
  }

  const chainFile = fs.readFileSync(
    `./ts-scripts/relayer/config/${env}/chains.json`
  );
  const chains = JSON.parse(chainFile.toString());
  if (chains.operatingChains) {
    operatingChains = chains.operatingChains;
  }
  if (!operatingChains) {
    return allChains;
  }

  const output: ChainInfo[] = [];
  operatingChains.forEach((x: number) => {
    const item = allChains.find((y) => {
      return x == y.chainId;
    });
    if (item) {
      output.push(item);
    }
  });

  return output;
}

export function loadChains(): ChainInfo[] {
  const chainFile = fs.readFileSync(
    `./ts-scripts/relayer/config/${env}/chains.json`
  );
  const chains = JSON.parse(chainFile.toString());
  if (!chains.chains) {
    throw Error("Failed to pull chain config file!");
  }
  return chains.chains;
}

export function getChain(chain: ChainId): ChainInfo {
  const chains = loadChains();
  const output = chains.find((x) => x.chainId == chain);
  if (!output) {
    throw Error("bad chain ID");
  }

  return output;
}

export function loadPrivateKey(): string {
  const privateKey = get_env_var("WALLET_KEY");
  if (!privateKey) {
    throw Error("Failed to find private key for this process!");
  }
  return privateKey;
}

export function loadGuardianSetIndex(): number {
  const chainFile = fs.readFileSync(
    `./ts-scripts/relayer/config/${env}/chains.json`
  );
  const chains = JSON.parse(chainFile.toString());
  if (chains.guardianSetIndex == undefined) {
    throw Error("Failed to pull guardian set index from the chains file!");
  }
  return chains.guardianSetIndex;
}

export function loadRelayProviders(): Deployment[] {
  const contractsFile = fs.readFileSync(
    `./ts-scripts/relayer/config/${env}/contracts.json`
  );
  if (!contractsFile) {
    throw Error("Failed to find contracts file for this process!");
  }
  const contracts = JSON.parse(contractsFile.toString());
  if (contracts.useLastRun || lastRunOverride) {
    const lastRunFile = fs.readFileSync(
      `./ts-scripts/relayer/output/${env}/deployRelayProvider/lastrun.json`
    );
    if (!lastRunFile) {
      throw Error(
        "Failed to find last run file for the deployRelayProvider process!"
      );
    }
    const lastRun = JSON.parse(lastRunFile.toString());
    return lastRun.relayProviderProxies;
  } else if (contracts.useLastRun == false) {
    return contracts.relayProviders;
  } else {
    throw Error("useLastRun was an invalid value from the contracts config");
  }
}

export function loadCoreRelayers(): Deployment[] {
  const contractsFile = fs.readFileSync(
    `./ts-scripts/relayer/config/${env}/contracts.json`
  );
  if (!contractsFile) {
    throw Error("Failed to find contracts file for this process!");
  }
  const contracts = JSON.parse(contractsFile.toString());
  if (contracts.useLastRun || lastRunOverride) {
    const lastRunFile = fs.readFileSync(
      `./ts-scripts/relayer/output/${env}/deployCoreRelayer/lastrun.json`
    );
    if (!lastRunFile) {
      throw Error("Failed to find last run file for the Core Relayer process!");
    }
    const lastRun = JSON.parse(lastRunFile.toString());
    return lastRun.coreRelayerProxies;
  } else {
    return contracts.coreRelayers;
  }
}

export function loadMockIntegrations(): Deployment[] {
  const contractsFile = fs.readFileSync(
    `./ts-scripts/relayer/config/${env}/contracts.json`
  );
  if (!contractsFile) {
    throw Error("Failed to find contracts file for this process!");
  }
  const contracts = JSON.parse(contractsFile.toString());
  if (contracts.useLastRun || lastRunOverride) {
    const lastRunFile = fs.readFileSync(
      `./ts-scripts/relayer/output/${env}/deployMockIntegration/lastrun.json`
    );
    if (!lastRunFile) {
      throw Error(
        "Failed to find last run file for the deploy mock integration process!"
      );
    }
    const lastRun = JSON.parse(lastRunFile.toString());
    return lastRun.mockIntegrations;
  } else {
    return contracts.mockIntegrations;
  }
}

export function loadGuardianKey(): string {
  const guardianKey = get_env_var("GUARDIAN_KEY");
  if (!guardianKey) {
    throw Error("Failed to find guardian key for this process!");
  }
  return guardianKey;
}

export function writeOutputFiles(output: any, processName: string) {
  fs.mkdirSync(`./ts-scripts/relayer/output/${env}/${processName}`, {
    recursive: true,
  });
  fs.writeFileSync(
    `./ts-scripts/relayer/output/${env}/${processName}/lastrun.json`,
    JSON.stringify(output),
    { flag: "w" }
  );
  fs.writeFileSync(
    `./ts-scripts/relayer/output/${env}/${processName}/${Date.now()}.json`,
    JSON.stringify(output),
    { flag: "w" }
  );
}

export function getSigner(chain: ChainInfo): Signer {
  let provider = getProvider(chain);
  let signer = new ethers.Wallet(loadPrivateKey(), provider);
  return signer;
}

export function getProvider(
  chain: ChainInfo
): ethers.providers.StaticJsonRpcProvider {
  let provider = new ethers.providers.StaticJsonRpcProvider(
    loadChains().find((x: any) => x.chainId == chain.chainId)?.rpc || ""
  );

  return provider;
}

export function getRelayProviderAddress(chain: ChainInfo): string {
  const thisChainsProvider = loadRelayProviders().find(
    (x: any) => x.chainId == chain.chainId
  )?.address;
  if (!thisChainsProvider) {
    throw new Error(
      "Failed to find a RelayProvider contract address on chain " +
        chain.chainId
    );
  }
  return thisChainsProvider;
}

export function getRelayProvider(
  chain: ChainInfo,
  provider?: ethers.providers.StaticJsonRpcProvider
): RelayProvider {
  const thisChainsProvider = getRelayProviderAddress(chain);
  const contract = RelayProvider__factory.connect(
    thisChainsProvider,
    provider || getSigner(chain)
  );
  return contract;
}

export function getCoreRelayerAddress(chain: ChainInfo): string {
  const thisChainsRelayer = loadCoreRelayers().find(
    (x: any) => x.chainId == chain.chainId
  )?.address;
  if (!thisChainsRelayer) {
    throw new Error(
      "Failed to find a CoreRelayer contract address on chain " + chain.chainId
    );
  }
  return thisChainsRelayer;
}

export function getCoreRelayer(
  chain: ChainInfo,
  provider?: ethers.providers.StaticJsonRpcProvider
): CoreRelayer {
  const thisChainsRelayer = getCoreRelayerAddress(chain);
  const contract = CoreRelayer__factory.connect(
    thisChainsRelayer,
    provider || getSigner(chain)
  );
  return contract;
}

export function getMockIntegrationAddress(chain: ChainInfo): string {
  const thisMock = loadMockIntegrations().find(
    (x: any) => x.chainId == chain.chainId
  )?.address;
  if (!thisMock) {
    throw new Error(
      "Failed to find a mock integration contract address on chain " +
        chain.chainId
    );
  }
  return thisMock;
}

export function getMockIntegration(chain: ChainInfo): MockRelayerIntegration {
  const thisIntegration = getMockIntegrationAddress(chain);
  const contract = MockRelayerIntegration__factory.connect(
    thisIntegration,
    getSigner(chain)
  );
  return contract;
}
