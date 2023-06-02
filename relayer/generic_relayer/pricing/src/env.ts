import * as fs from "fs/promises";
import yargs from "yargs";
import {
  Environment,
  ProvidersOpts,
  RedisOptions,
  StandardRelayerAppOpts,
} from "relayer-engine";
import {
  CHAIN_ID_ETH,
  CHAIN_ID_BSC,
  EVMChainId,
  ChainId,
} from "@certusone/wormhole-sdk";
import { rootLogger } from "@wormhole-foundation/offchain-generic-relayer/src/log";
import { ethers } from "ethers";
import { PricingContext } from "./app";

const SCRIPTS_DIR = "../../../ethereum/ts-scripts/relayer";

type Opts = {
  flag: Flag;
};

enum Flag {
  TiltKub = "tiltkub",
  Tilt = "tilt",
  Testnet = "testnet",
  K8sTestnet = "k8s-testnet",
  Mainnet = "mainnet",
}

type ContractConfigEntry = { chainId: EVMChainId; address: "string" };
type ContractsJson = {
  deliveryProviders: ContractConfigEntry[];
  coreRelayers: ContractConfigEntry[];
  mockIntegrations: ContractConfigEntry[];
};

export interface PricingMonitorAppConfig {
  contractsJsonPath: string;
  name: string;
  wormholeRpcs: [string];
  providers?: ProvidersOpts;
  fetchSourceTxhash: boolean;
  logLevel: string;
  logFormat: "json" | "text";
}

const defaults: { [key in Flag]: PricingMonitorAppConfig } = {
  [Flag.TiltKub]: {
    name: "PricingMonitor",
    contractsJsonPath: `${SCRIPTS_DIR}/config/${Flag.Tilt}/contracts.json`,
    logLevel: "debug",
    logFormat: "text",
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
    fetchSourceTxhash: false,
  },
  [Flag.Tilt]: {
    name: "GenericRelayer",
    contractsJsonPath: `${SCRIPTS_DIR}/config/${Flag.Tilt}/contracts.json`,
    logLevel: "debug",
    logFormat: "text",
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
    fetchSourceTxhash: false,
  },
  // TODO
  [Flag.K8sTestnet]: {
    contractsJsonPath: `${SCRIPTS_DIR}/config/${Flag.Testnet}/contracts.json`,
    name: "GenericRelayer",
    logLevel: "debug",
    logFormat: "json",
    spyEndpoint: "spy:7073",
    wormholeRpcs: ["https://wormhole-v2-testnet-api.certus.one"],
    fetchSourceTxhash: true,
    redisCluster: {
      redis: { host: "redis", port: 6379 },
    },
  } as any,
  [Flag.Testnet]: {
    contractsJsonPath: `${SCRIPTS_DIR}/config/${Flag.Testnet}/contracts.json`,
    name: "GenericRelayer",
    logLevel: "debug",
    logFormat: "text",
    wormholeRpcs: ["https://wormhole-v2-testnet-api.certus.one"],
    fetchSourceTxhash: true,
  },
  [Flag.Mainnet]: {} as any,
};

export async function loadAppConfig(): Promise<{
  env: Environment;
  opts: PricingMonitorAppConfig & StandardRelayerAppOpts;
  deliveryProviders: Record<EVMChainId, string>;
  wormholeRelayers: Record<EVMChainId, string>;
}> {
  const { flag } = getEnvironmentOptions();
  const config = await loadAndMergeConfig(flag);
  const contracts = await loadJson<ContractsJson>(config.contractsJsonPath);

  const deliveryProviders = {} as Record<EVMChainId, string>;
  const wormholeRelayers = {} as Record<EVMChainId, string>;
  contracts.deliveryProviders.forEach(
    ({ chainId, address }: ContractConfigEntry) =>
      (deliveryProviders[chainId] = ethers.utils.getAddress(address))
  );
  contracts.coreRelayers.forEach(
    ({ chainId, address }: ContractConfigEntry) =>
      (wormholeRelayers[chainId] = ethers.utils.getAddress(address))
  );

  return {
    deliveryProviders,
    wormholeRelayers,
    env: flagToEnvironment(flag),
    opts: {
      ...config,
      logger: rootLogger(config.logLevel, config.logFormat),
      privateKeys: privateKeys(contracts),
    },
  };
}

function getEnvironmentOptions(): Opts {
  let opts = yargs(process.argv.slice(2)).argv as unknown as Opts;
  if (opts.flag == undefined) {
    opts.flag = process.env.GR_RE_FLAG as Flag;
  }
  if (!validateStringEnum(Flag, opts.flag)) {
    throw new Error("Unrecognized flag variant: " + opts.flag);
  }
  return opts;
}

function loadAndMergeConfig(flag: Flag): PricingMonitorAppConfig {
  const base = defaults[flag];
  const isRedisCluster = !!process.env.REDIS_CLUSTER_ENDPOINTS;
  return {
    name: process.env.GENERIC_RELAYER_NAME || base.name,
    // env: process.env.NODE_ENV?.trim()?.toLowerCase() || "local",
    contractsJsonPath:
      process.env.CONTRACTS_JSON_PATH || base.contractsJsonPath,
    logFormat: (process.env.LOG_FORMAT as "text" | "json") || base.logFormat,
    logLevel: process.env.LOG_LEVEL || base.logLevel,
    wormholeRpcs: process.env.WORMHOLE_RPCS
      ? JSON.parse(process.env.WORMHOLE_RPCS)
      : base.wormholeRpcs,
    providers: process.env.BLOCKCHAIN_PROVIDERS
      ? JSON.parse(process.env.BLOCKCHAIN_PROVIDERS)
      : base.providers,
    fetchSourceTxhash: process.env.FETCH_SOURCE_TX_HASH
      ? JSON.parse(process.env.FETCH_SOURCE_TX_HASH)
      : base.fetchSourceTxhash,

    // concurrency: Number(process.env.RELAY_CONCURRENCY) || 5,
    // influx: {
    //   url: process.env.INFLUXDB_URL,
    //   org: process.env.INFLUXDB_ORG,
    //   bucket: process.env.INFLUXDB_BUCKET,
    //   token: process.env.INFLUXDB_TOKEN,
    // },
  };
}

function privateKeys(contracts: ContractsJson): {
  [k in Partial<EVMChainId>]: string[];
} {
  const chainIds = new Set(contracts.coreRelayers.map((r) => r.chainId));
  let privateKeysArray = [] as string[];
  if (process.env.EVM_PRIVATE_KEYS) {
    privateKeysArray = JSON.parse(process.env.EVM_PRIVATE_KEYS);
  } else if (process.env.EVM_PRIVATE_KEY) {
    privateKeysArray = [process.env.EVM_PRIVATE_KEY];
  } else if (process.env.PRIVATE_KEY) {
    // tilt
    privateKeysArray = [process.env.PRIVATE_KEY];
  } else {
    // Todo: remove this
    // tilt evm private key
    console.log(
      "Warning: using tilt private key because no others were specified"
    );
    privateKeysArray = [
      "6cbed15c793ce57650b9877cf6fa156fbef513c4e6134f022a85b1ffdd59b2a1",
    ];
  }
  const privateKeys = {} as Record<EVMChainId, string[]>;
  for (const chainId of chainIds) {
    privateKeys[chainId] = privateKeysArray;
  }
  return privateKeys;
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

function validateStringEnum<O extends Object>(
  enumObject: O,
  passed: string
): boolean {
  for (const value of Object.values(enumObject)) {
    if (value === passed) {
      return true;
    }
  }
  return false;
}

function loadJson<T>(path: string): Promise<T> {
  return fs
    .readFile(path, {
      encoding: "utf-8",
    })
    .then(JSON.parse) as Promise<T>;
}

//TODO there must be a better way to do this
export function getEthersProvider(
  ctx: PricingContext,
  chainId: ChainId
): ethers.providers.JsonRpcProvider {
  const rpc = ctx.providers.evm[chainId as EVMChainId];
  if (rpc == undefined || rpc.length == 0) {
    throw new Error(`No rpc found for chainId ${chainId}`);
  }

  return rpc[0];
}

//TODO there must be a better way to do this
export function getAllChains(ctx: PricingContext): ChainId[] {
  const records = ctx.deliveryProviders;
  return Object.keys(records).map((k) => parseInt(k) as ChainId); //Should be safe
}
