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
import { loadAppConfig } from "./env";

export type GRContext = StandardRelayerContext & {
  relayProviders: Record<EVMChainId, string>;
  wormholeRelayers: Record<EVMChainId, string>;
};

async function main() {
  const { env, opts, relayProviders, wormholeRelayers } = await loadAppConfig();
  const app = new StandardRelayerApp<GRContext>(env, opts);

  // Set up middleware
  app.use(async (ctx: GRContext, next: Next) => {
    ctx.relayProviders = deepCopy(relayProviders);
    ctx.wormholeRelayers = deepCopy(wormholeRelayers);
    next();
  });

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

main().catch((e) => {
  console.error("Encountered unrecoverable error:");
  console.error(e);
  process.exit(1);
});
