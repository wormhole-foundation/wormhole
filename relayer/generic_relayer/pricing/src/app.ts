import Koa from "koa";
import Router from "koa-router";
import {
  Next,
  RelayerApp,
  StandardRelayerAppOpts,
  StandardRelayerContext,
  logging,
  wallets,
  missedVaas,
  providers,
  sourceTx,
} from "relayer-engine";
import { EVMChainId } from "@certusone/wormhole-sdk";
import { processProviderPriceUpdate } from "./processor";
import { Logger } from "winston";
import deepCopy from "clone";
import { loadAppConfig } from "@wormhole-foundation/offchain-generic-relayer/src/env";

export type PricingContext = StandardRelayerContext & {
  relayProviders: Record<EVMChainId, string>;
  wormholeRelayers: Record<EVMChainId, string>;
  opts: StandardRelayerAppOpts;
};

async function main() {
  const { env, opts, relayProviders, wormholeRelayers } = await loadAppConfig();
  const logger = opts.logger!;
  logger.debug("Redis config: ", opts.redis);

  const app = new RelayerApp<PricingContext>(env, opts);
  const { privateKeys, name, wormholeRpcs } = opts;

  app.logger(logger);
  app.use(logging(logger));
  app.use(providers(opts.providers));
  if (opts.privateKeys && Object.keys(opts.privateKeys).length) {
    app.use(
      wallets(env, {
        logger,
        namespace: name,
        privateKeys: privateKeys!,
        metrics: { registry: app.metricsRegistry },
      })
    );
  }
  if (opts.fetchSourceTxhash) {
    app.use(sourceTx());
  }

  // Set up middleware
  app.use(async (ctx: PricingContext, next: Next) => {
    ctx.relayProviders = deepCopy(relayProviders);
    ctx.wormholeRelayers = deepCopy(wormholeRelayers);
    ctx.opts = { ...opts };
    next();
  });

  // Set up routes
  app.multiple(deepCopy(wormholeRelayers), processProviderPriceUpdate);

  app.listen();
  runApi(app, opts, logger);
}

function runApi(relayer: any, { port, redis }: any, logger: Logger) {
  const app = new Koa();
  const router = new Router();

  router.get("/metrics", async (ctx: Koa.Context) => {
    ctx.body = await relayer.metricsRegistry?.metrics();
  });

  app.use(router.routes());
  app.use(router.allowedMethods());

  if (redis?.host) {
    app.use(relayer.storageKoaUI("/ui"));
  }

  port = Number(port) || 3000;
  app.listen(port, () => {
    logger.info(`Running on ${port}...`);
    logger.info(`For the UI, open http://localhost:${port}/ui`);
    logger.info(
      `For prometheus metrics, open http://localhost:${port}/metrics`
    );
    logger.info("Make sure Redis is running on port 6379 by default");
  });
}

main().catch((e) => {
  console.error("Encountered unrecoverable error:");
  console.error(e);
  process.exit(1);
});
