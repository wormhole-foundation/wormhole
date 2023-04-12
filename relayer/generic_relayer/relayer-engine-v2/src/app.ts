import * as fs from "fs/promises";
import yargs from "yargs";
import * as Koa from "koa";
import {
  Environment,
  Next,
  StandardRelayerApp,
  StandardRelayerAppOpts,
  StandardRelayerContext,
} from "relayer-engine";
import {
  CHAIN_ID_ETH,
  CHAIN_ID_BSC,
  EVMChainId,
  tryNativeToHexString,
} from "@certusone/wormhole-sdk";
import { processGenericRelayerVaa } from "./processor";
import { Logger } from "winston";
import * as deepCopy from "clone";
import { GRRelayerAppConfig, loadAppConfig } from "./env";
import { dbg } from "../pkgs/sdk/src";

export type GRContext = StandardRelayerContext & {
  relayProviders: Record<EVMChainId, string>;
  wormholeRelayers: Record<EVMChainId, string>;
  opts: StandardRelayerAppOpts;
};

async function main() {
  const { env, opts, relayProviders, wormholeRelayers } = await loadAppConfig();
  // gets mangled by app constructor somehow...
  const logger = opts.logger!;

  const app = new StandardRelayerApp<GRContext>(env, opts);

  // Set up middleware
  app.use(async (ctx: GRContext, next: Next) => {
    ctx.relayProviders = deepCopy(relayProviders);
    ctx.wormholeRelayers = deepCopy(wormholeRelayers);
    ctx.opts = deepCopy(opts);
    next();
  });

  // Set up routes
  app.multiple(deepCopy(wormholeRelayers), processGenericRelayerVaa);

  app.listen();
  runUI(app, opts, logger);
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
