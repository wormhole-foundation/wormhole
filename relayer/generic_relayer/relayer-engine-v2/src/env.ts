import * as fs from "fs/promises";
import yargs from "yargs";
import {
  Environment,
  ProvidersOpts,
  RedisOptions,
  StandardRelayerAppOpts,
} from "wormhole-relayer";
import {
  CHAIN_ID_ETH,
  CHAIN_ID_BSC,
  EVMChainId,
} from "@certusone/wormhole-sdk";
import { ClusterOptions } from "ioredis";

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
  relayProviders: ContractConfigEntry[];
  coreRelayers: ContractConfigEntry[];
  mockIntegrations: ContractConfigEntry[];
};

interface GRRelayerAppConfig {
  contractsJsonPath: string;
  name: string;
  spyEndpoint: string;
  wormholeRpcs: [string];
  providers: ProvidersOpts;
  fetchSourceTxhash: boolean;
  logLevel: string;
  redis: RedisOptions;
  redisCluster?: StandardRelayerAppOpts["redisCluster"];
  redisClusterEndpoints?: StandardRelayerAppOpts["redisClusterEndpoints"];
}

const defaults: { [key in Flag]: GRRelayerAppConfig } = {
  [Flag.TiltKub]: {
    name: "GenericRelayer",
    contractsJsonPath: `${SCRIPTS_DIR}/config/${Flag.TiltKub}/contracts.json`,
    spyEndpoint: "spy:7072",
    logLevel: "debug",
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
    redis: { host: "redis", port: 6379 },
  },
  [Flag.Tilt]: {
    name: "GenericRelayer",
    contractsJsonPath: `${SCRIPTS_DIR}/config/${Flag.Tilt}/contracts.json`,
    logLevel: "debug",
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
    fetchSourceTxhash: false,
    redis: {},
  },
  [Flag.K8sTestnet]: {} as any,
  [Flag.Testnet]: {} as any,
  [Flag.Mainnet]: {} as any,
};

// async function loadAndMergeConfig(flag: Flag): Promise<GRRelayerAppConfig> {
//   const file = await fs.readFile(`./configs/${flag}.json`, {
//     encoding: "utf-8",
//   });
//   const config = JSON.parse(file);
//   return mergeDeep({}, [defaults[flag] ?? {}, config]) as GRRelayerAppConfig;
// }

export async function loadAppConfig(): Promise<{
  env: Environment;
  opts: StandardRelayerAppOpts;
  relayProviders: Record<EVMChainId, string>;
  wormholeRelayers: Record<EVMChainId, string>;
}> {
  const { flag } = getEnvironmentOptions();
  const config = await loadAndMergeConfig(flag);
  const contracts = await loadJson<ContractsJson>(config.contractsJsonPath);

  const relayProviders = {} as Record<EVMChainId, string>;
  const wormholeRelayers = {} as Record<EVMChainId, string>;
  contracts.relayProviders.forEach(
    ({ chainId, address }: ContractConfigEntry) =>
      (relayProviders[chainId] = address)
  );
  contracts.coreRelayers.forEach(
    ({ chainId, address }: ContractConfigEntry) =>
      (wormholeRelayers[chainId] = address)
  );

  return {
    relayProviders,
    wormholeRelayers,
    env: flagToEnvironment(flag),
    opts: {
      ...config,
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

function loadAndMergeConfig(flag: Flag): GRRelayerAppConfig {
  const base = defaults[flag];
  const isRedisCluster = !!process.env.REDIS_CLUSTER_ENDPOINTS;
  return {
    name: process.env.GENERIC_RELAYER_NAME || base.name,
    // env: process.env.NODE_ENV?.trim()?.toLowerCase() || "local",
    contractsJsonPath:
      process.env.CONTRACTS_JSON_PATH || base.contractsJsonPath,
    logLevel: process.env.LOG_LEVEL || base.logLevel,
    spyEndpoint: process.env.SPY_URL || base.spyEndpoint,
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

    redisClusterEndpoints: process.env.REDIS_CLUSTER_ENDPOINTS?.split(","), // "url1:port,url2:port"
    redisCluster: isRedisCluster
      ? <ClusterOptions>{
          dnsLookup: (address: any, callback: any) => callback(null, address),
          slotsRefreshTimeout: 1000,
          redisOptions: {
            tls: process.env.REDIS_TLS ? {} : undefined,
            username: process.env.REDIS_USERNAME,
            password: process.env.REDIS_PASSWORD,
          },
        }
      : undefined,
    redis: <RedisOptions>{
      tls: process.env.REDIS_TLS ? {} : undefined,
      host: process.env.REDIS_HOST ? undefined : process.env.REDIS_HOST,
      port: process.env.REDIS_CLUSTER_ENDPOINTS
        ? undefined
        : Number(process.env.REDIS_PORT) || undefined,
      username: process.env.REDIS_USERNAME,
      password: process.env.REDIS_PASSWORD,
    },
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
