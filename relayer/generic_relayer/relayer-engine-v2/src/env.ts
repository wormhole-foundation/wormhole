import * as fs from "fs/promises";
import yargs from "yargs";
import * as Koa from "koa";
import {
  Environment,
  Next,
  StandardRelayerApp,
  StandardRelayerContext,
} from "wormhole-relayer";
import { defaultLogger } from "wormhole-relayer/lib/logging";
import {
  CHAIN_ID_ETH,
  CHAIN_ID_BSC,
  EVMChainId,
  tryNativeToHexString,
} from "@certusone/wormhole-sdk";
import { rootLogger } from "./log";
import { processGenericRelayerVaa } from "./processor";
import { Logger } from "winston";
import * as deepCopy from "clone";

export type Opts = {
  flag: Flag;
};

export enum Flag {
  TiltKub = "tiltkub",
  Tilt = "tilt",
  Testnet = "testnet",
  K8sTestnet = "k8s-testnet",
  Mainnet = "mainnet",
}

export type ContractConfigEntry = { chainId: EVMChainId; address: "string" };
export type ContractsJson = {
  relayProviders: ContractConfigEntry[];
  coreRelayers: ContractConfigEntry[];
  mockIntegrations: ContractConfigEntry[];
};

export function getEnvironmentOptions() {
  let opts = yargs(process.argv.slice(2)).argv as unknown as Opts;
  return opts;
}

let initialized = false;
let contracts: ContractsJson;
export async function init() {
  const opts = getEnvironmentOptions();
  contracts = await loadContractsJson(opts.flag);
  initialized = true;
}
function uninitialized() {
  throw new Error("init function was not called.");
}

export function getAppConfig() {
  const contracts = getContractsJson();
  const options = getEnvironmentOptions();
  if (options.flag == Flag.TiltKub) {
    return {
      name: "GenericRelayer",
      privateKeys: privateKeys(contracts),
      spyEndpoint: "spy:7072",
      wormholeRpcs: ["http://guardian:7071"],
      providers: {
        chains: {
          [CHAIN_ID_ETH]: {
            endpoints: ["http://eth-devnet:8545/"],
          },
          [CHAIN_ID_BSC]: {
            endpoints: ["http://eth-devnet2:8545/"],
          },
        },
      },
      logger: defaultLogger,
      fetchSourceTxhash: false,
      redis: { host: "redis", port: 6379 },
      // redisCluster: {},
      // redisClusterEndpoints: [],
    };

    //else assume localhost / tilt
  } else {
    return {
      name: "GenericRelayer",
      privateKeys: privateKeys(contracts),
      spyEndpoint: "localhost:7072",
      wormholeRpcs: ["http://localhost:7071"],
      providers: {
        chains: {
          [CHAIN_ID_ETH]: {
            endpoints: ["http://localhost:8545/"],
          },
          [CHAIN_ID_BSC]: {
            endpoints: ["http://localhost:8546/"],
          },
        },
      },
      logger: defaultLogger,
      fetchSourceTxhash: false,
      redis: {},
      // redisCluster: {},
      // redisClusterEndpoints: [],
    };
  }
}

//internal only
async function loadContractsJson(flag: Flag): Promise<ContractsJson> {
  if ((flag = Flag.TiltKub)) {
    flag = Flag.Tilt; //TiltKub contracts are the same as tilt
  }
  return JSON.parse(
    await fs.readFile(`${SCRIPTS_DIR}/config/${flag}/contracts.json`, {
      encoding: "utf-8",
    })
  ) as ContractsJson;
}

export function getContractsJson(): ContractsJson {
  if (!initialized || !contracts) {
    uninitialized();
  }
  return contracts;
}

function privateKeys(contracts: ContractsJson) {
  const chainIds = new Set(contracts.coreRelayers.map((r) => r.chainId));
  //TODO not this
  const privateKey =
    "6cbed15c793ce57650b9877cf6fa156fbef513c4e6134f022a85b1ffdd59b2a1"; //private key 1 for tilt //process.env["PRIVATE_KEY"]! as string;
  const privateKeys = {} as Record<EVMChainId, [string]>;
  for (const chainId of chainIds) {
    privateKeys[chainId] = [privateKey];
  }
  return privateKeys;
}

export function getEnvironment() {
  const options = getEnvironmentOptions();
  return flagToEnvironment(options.flag);
}

function flagToEnvironment(flag: Flag): Environment {
  switch (flag) {
    case Flag.K8sTestnet:
      return Environment.TESTNET;
    case Flag.Testnet:
      return Environment.TESTNET;
    case Flag.Mainnet:
      return Environment.MAINNET;
    case Flag.Tilt:
      return Environment.DEVNET;
    case Flag.TiltKub:
      return Environment.DEVNET;
  }
}

const SCRIPTS_DIR = "../../../ethereum/ts-scripts/relayer";
