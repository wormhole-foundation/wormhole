import { ChainInfo, getOperatingChains, init } from "../helpers/env";
import { sendMessage } from "./messageUtils";
import { register, Counter } from "prom-client";
import Koa from "koa";
import Router from "koa-router";

init();
const chains = getOperatingChains();

const sentCounter = new Counter({
  name: "sent_messages",
  help: "Number of messages sent",
  labelNames: ["from", "to"],
  registers: [register],
});

const sentSuccessCounter = new Counter({
  name: "sent_messages_success",
  help: "Number of messages sent successfully",
  labelNames: ["from", "to"],
  registers: [register],
});

const sentFailedCounter = new Counter({
  name: "sent_messages_failed",
  help: "Number of messages failed to relay within 40 seconds",
  labelNames: ["from", "to"],
  registers: [register],
});

const failedToSendCounter = new Counter({
  name: "failed_to_send_messages",
  help: "Number of messages failed to send",
  labelNames: ["from", "to"],
  registers: [register],
});

async function main() {
  console.log(process.argv);
  console.log(chains);

  let period = 15 * 60 * 1000;
  if (tryGetArg("--minutes")) {
    period = Number(getArg("--minutes")) * 60 * 1000;
  } else if (tryGetArg("--seconds")) {
    period = Number(getArg("--seconds")) * 1000;
  }
  console.log(
    `Running test every ${period / 1000} seconds (${period /
      1000 /
      60}) minutes)`
  );

  runMetricsServer({ port: 1234 });

  while (true) {
    console.log("Running test...");
    await run();

    console.log("Sleeping...");
    await new Promise((resolve) => setTimeout(resolve, period));
  }
}

async function run() {
  if (tryGetArg("--from") && tryGetArg("--to")) {
    await sendMessageAndEmitMetrics(
      getChainById(getArg("--from")!),
      getChainById(getArg("--to")!)
    );
  } else if (tryGetArg("--per-chain")) {
    for (let i = 0; i < chains.length; ++i) {
      await sendMessageAndEmitMetrics(
        chains[i],
        chains[i === 0 ? chains.length - 1 : 0]
      );
    }
  } else if (tryGetArg("--matrix")) {
    for (let i = 0; i < chains.length; ++i) {
      for (let j = 0; i < chains.length; ++i) {
        await sendMessageAndEmitMetrics(chains[i], chains[j]);
      }
    }
  }
  await sendMessageAndEmitMetrics(chains[0], chains[1]);
}

async function sendMessageAndEmitMetrics(from: ChainInfo, to: ChainInfo) {
  try {
    const didRelay = await sendMessage(from, to);
    sentCounter.inc({ from: from.chainId, to: to.chainId });

    (didRelay ? sentSuccessCounter : sentFailedCounter).inc({
      from: from.chainId,
      to: to.chainId,
    });
  } catch (e) {
    failedToSendCounter.inc({ from: from.chainId, to: to.chainId });
    console.error(
      `Failed to send message from ${from.chainId} to ${to.chainId}`
    );
    console.error(e);
  }
}

function getChainById(id: number | string): ChainInfo {
  id = Number(id);
  const chain = chains.find((c) => c.chainId === id);
  if (!chain) {
    throw new Error("chainId not found, " + id);
  }
  return chain;
}

function runMetricsServer({ port }: any) {
  const app = new Koa();
  const router = new Router();

  router.get("/metrics", async (ctx: Koa.Context) => {
    console.log("Metrics endpoint hit");
    console.log(
      `Metrics: ${JSON.stringify(
        await register.getMetricsAsJSON(),
        undefined,
        2
      )}`
    );
    ctx.body = await register.getMetricsAsJSON();
  });

  app.use(router.routes());
  app.use(router.allowedMethods());

  port = Number(port) || 3000;
  app.listen(port, () => {
    console.info(`Running on ${port}...`);
    console.info(
      `For prometheus metrics, open http://localhost:${port}/metrics`
    );
  });
}

function tryGetArg(pattern: string | string[]): string | undefined {
  return getArg(pattern, { required: false });
}

function getArg(
  patterns: string | string[],
  {
    isFlag = false,
    required = true,
  }: { isFlag?: boolean; required?: boolean } = {
    isFlag: false,
    required: true,
  }
): string | undefined {
  let idx: number = -1;
  if (typeof patterns === "string") {
    patterns = [patterns];
  }
  for (const pattern of patterns) {
    idx = process.argv.findIndex((x) => x === pattern);
    if (idx !== -1) {
      break;
    }
  }
  if (idx === -1) {
    if (required) {
      throw new Error(
        "Missing required cmd line arg: " + JSON.stringify(patterns)
      );
    }
    return undefined;
  }
  if (isFlag) {
    return process.argv[idx];
  }
  return process.argv[idx + 1];
}

console.log("Start!");
main().then(() => console.log("Done!"));
