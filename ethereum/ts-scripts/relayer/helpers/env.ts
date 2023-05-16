import { ChainId } from "@certusone/wormhole-sdk";
import { ethers } from "ethers";
import fs from "fs";

import { CoreRelayer } from "../../../ethers-contracts";
import { RelayProvider } from "../../../ethers-contracts";
import { MockRelayerIntegration } from "../../../ethers-contracts";

import { RelayProvider__factory } from "../../../ethers-contracts";
import { CoreRelayer__factory } from "../../../ethers-contracts";
import { MockRelayerIntegration__factory } from "../../../ethers-contracts";
import {
  Create2Factory,
  Create2Factory__factory,
} from "../../../ethers-contracts";
import { proxyContractSalt, setupContractSalt } from "./deployments";

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

export function loadCreate2Factories(): Deployment[] {
  const contractsFile = fs.readFileSync(
    `./ts-scripts/relayer/config/${env}/contracts.json`
  );
  if (!contractsFile) {
    throw Error("Failed to find contracts file for this process!");
  }
  const contracts = JSON.parse(contractsFile.toString());
  if (contracts.useLastRun || lastRunOverride) {
    const lastRunFile = fs.readFileSync(
      `./ts-scripts/relayer/output/${env}/deployCreate2Factory/lastrun.json`
    );
    if (!lastRunFile) {
      throw Error(
        "Failed to find last run file for the deployCreate2Factory process!"
      );
    }
    const lastRun = JSON.parse(lastRunFile.toString());
    return lastRun.create2Factories;
  } else {
    return contracts.create2Factories;
  }
}

//TODO load these keys more intelligently,
//potentially from devnet-consts.
//Also, make sure the signers are correctly ordered by index,
//As the index gets encoded into the signature.
export function loadGuardianKeys(): string[] {
  const output = [];
  const NUM_GUARDIANS = get_env_var("NUM_GUARDIANS");
  const guardianKey = get_env_var("GUARDIAN_KEY");
  const guardianKey2 = get_env_var("GUARDIAN_KEY2");

  let numGuardians: number = 0;
  console.log("NUM_GUARDIANS variable : " + NUM_GUARDIANS);

  if (!NUM_GUARDIANS) {
    numGuardians = 1;
  } else {
    numGuardians = parseInt(NUM_GUARDIANS);
  }

  if (!guardianKey) {
    throw Error("Failed to find guardian key for this process!");
  }
  output.push(guardianKey);

  if (numGuardians >= 2) {
    if (!guardianKey2) {
      throw Error("Failed to find guardian key 2 for this process!");
    }
    output.push(guardianKey2);
  }

  return output;
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

export function getSigner(chain: ChainInfo): ethers.Wallet {
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

export function loadGuardianRpc(): string {
  const chainFile = fs.readFileSync(
    `./ts-scripts/relayer/config/${env}/chains.json`
  );
  if (!chainFile) {
    throw Error("Failed to find contracts file for this process!");
  }
  const chain = JSON.parse(chainFile.toString());
  return chain.guardianRPC;
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

const coreRelayerAddressesCache: Partial<Record<ChainId, string>> = {};
export async function getCoreRelayerAddress(
  chain: ChainInfo,
  forceCalculate?: boolean
): Promise<string> {
  const contractsFile = fs.readFileSync(
    `./ts-scripts/relayer/config/${env}/contracts.json`
  );
  if (!contractsFile) {
    throw Error("Failed to find contracts file for this process!");
  }
  const contracts = JSON.parse(contractsFile.toString());
  //If useLastRun is false, then we want to bypass the calculations and just use what the contracts file says.
  if (!contracts.useLastRun && !lastRunOverride && !forceCalculate) {
    const thisChainsRelayer = loadCoreRelayers().find(
      (x: any) => x.chainId == chain.chainId
    )?.address;
    if (thisChainsRelayer) {
      return thisChainsRelayer;
    } else {
      throw Error(
        "Failed to find a CoreRelayer contract address on chain " +
          chain.chainId
      );
    }
  }

  if (!coreRelayerAddressesCache[chain.chainId]) {
    const create2Factory = getCreate2Factory(chain);
    const signer = getSigner(chain).address;

    coreRelayerAddressesCache[
      chain.chainId
    ] = await create2Factory.computeProxyAddress(signer, proxyContractSalt);
  }

  return coreRelayerAddressesCache[chain.chainId]!;
}

export async function getCoreRelayer(
  chain: ChainInfo,
  provider?: ethers.providers.StaticJsonRpcProvider
): Promise<CoreRelayer> {
  const thisChainsRelayer = await getCoreRelayerAddress(chain);
  return CoreRelayer__factory.connect(
    thisChainsRelayer,
    provider || getSigner(chain)
  );
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

export function getCreate2FactoryAddress(chain: ChainInfo): string {
  const address = loadCreate2Factories().find(
    (x: any) => x.chainId == chain.chainId
  )?.address;
  if (!address) {
    throw new Error(
      "Failed to find a create2Factory contract address on chain " +
        chain.chainId
    );
  }
  return address;
}

export const getCreate2Factory = (chain: ChainInfo): Create2Factory =>
  Create2Factory__factory.connect(
    getCreate2FactoryAddress(chain),
    getSigner(chain)
  );
