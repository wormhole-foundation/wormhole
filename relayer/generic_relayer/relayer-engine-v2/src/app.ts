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

export type GRContext = StandardRelayerContext & {
  relayProviders: Record<EVMChainId, string>;
  wormholeRelayers: Record<EVMChainId, string>;
};

type Opts = {
  flag: Flag;
};

enum Flag {
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

const SCRIPTS_DIR = "../../../ethereum/ts-scripts/relayer";

async function main() {
  let opts = yargs(process.argv.slice(2)).argv as unknown as Opts;
  const contracts = await loadContractsJson(opts.flag);

  const app = new StandardRelayerApp<GRContext>(flagToEnvironment(opts.flag), {
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
  });

  // Build contract address maps
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

  // Set up middleware
  app.use(async (ctx: GRContext, next: Next) => {
    ctx.relayProviders = deepCopy(relayProviders);
    ctx.wormholeRelayers = deepCopy(wormholeRelayers);
    next();
  });

  // app
  //   .chain(CHAIN_ID_BSC)
  //   .address(
  //     "0x0eb0dd3aa41bd15c706bc09bc03c002b7b85aeac",
  //     processGenericRelayerVaa
  //   );

  // Set up routes
  app.multiple(deepCopy(wormholeRelayers), processGenericRelayerVaa);

  app.listen();
  runUI(app, opts, rootLogger);
}

function runUI(relayer: any, { port }: any, logger: Logger) {
  const app = new Koa();

  app.use(relayer.storageKoaUI("/ui"));

  port = Number(port) || 3000;
  app.listen(port, () => {
    logger.info(`Running on ${port}...`);
    logger.info(`For the UI, open http://localhost:${port}/ui`);
    logger.info("Make sure Redis is running on port 6379 by default");
  });
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
  }
}

async function loadContractsJson(flag: Flag): Promise<ContractsJson> {
  return JSON.parse(
    await fs.readFile(`${SCRIPTS_DIR}/config/${flag}/contracts.json`, {
      encoding: "utf-8",
    })
  ) as ContractsJson;
}

function privateKeys(contracts: ContractsJson) {
  const chainIds = new Set(contracts.coreRelayers.map((r) => r.chainId));
  const privateKey =
    "6cbed15c793ce57650b9877cf6fa156fbef513c4e6134f022a85b1ffdd59b2a1"; //private key 1 for tilt //process.env["PRIVATE_KEY"]! as string;
  const privateKeys = {} as Record<EVMChainId, [string]>;
  for (const chainId of chainIds) {
    privateKeys[chainId] = [privateKey];
  }
  return privateKeys;
}

main().catch((e) => {
  console.error("Encountered unrecoverable error:");
  console.error(e);
  process.exit(1);
});
