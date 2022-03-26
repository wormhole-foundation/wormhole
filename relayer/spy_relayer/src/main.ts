//This has to run first so that the process variables are set up when the other modules are instantiated.
require("./helpers/loadConfig");

import { setDefaultWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import { getCommonEnvironment } from "./configureEnv";
import { getLogger } from "./helpers/logHelper";
import { PromHelper, PromMode } from "./helpers/promHelpers";
import * as redisHelper from "./helpers/redisHelper";
import * as restListener from "./listener/rest_listen";
import * as spyListener from "./listener/spy_listen";
import * as relayWorker from "./relayer/relay_worker";

export enum ProcessType {
  LISTEN_ONLY = "--listen_only",
  RELAY_ONLY = "--relay_only",
  SPY_AND_RELAY = "spy and relay",
}

setDefaultWasm("node");
const logger = getLogger();

// Load the relay config data.
let runListen: boolean = true;
let runWorker: boolean = true;
let runRest: boolean = true;
let foundOne: boolean = false;
let error: string = "";

for (let idx = 0; idx < process.argv.length; ++idx) {
  if (process.argv[idx] === "--listen_only") {
    if (foundOne) {
      logger.error('May only specify one of "--listen_only" or "--relay_only"');
      error = "Multiple args found of --listen_only and --relay_only";
      break;
    }

    logger.info("spy_relay is running in listen only mode");
    runWorker = false;
    foundOne = true;
  }

  if (process.argv[idx] === "--relay_only") {
    if (foundOne) {
      logger.error(
        'May only specify one of "--listen_only", "--relay_only" or "--rest_only"'
      );
      error = "Multiple args found of --listen_only and --relay_only";
      break;
    }

    logger.info("spy_relay is running in relay only mode");
    runListen = false;
    runRest = false;
    foundOne = true;
  }
}

if (!foundOne) {
  logger.info("spy_relay is running both the listener and relayer");
}

if (
  !error &&
  spyListener.init(runListen) &&
  relayWorker.init(runWorker) &&
  restListener.init(runRest)
) {
  const commonEnv = getCommonEnvironment();
  const { promPort, readinessPort } = commonEnv;
  logger.info("prometheus client listening on port " + promPort);
  let promClient: PromHelper;
  const runBoth: boolean = runListen && runWorker;
  if (runBoth) {
    promClient = new PromHelper("spy_relay", promPort, PromMode.Both);
  } else if (runListen) {
    promClient = new PromHelper("spy_relay", promPort, PromMode.Listen);
  } else if (runWorker) {
    promClient = new PromHelper("spy_relay", promPort, PromMode.Relay);
  } else {
    logger.error("Invalid run mode for Prometheus");
    promClient = new PromHelper("spy_relay", promPort, PromMode.Both);
  }

  redisHelper.init(promClient);

  if (runListen) spyListener.run(promClient);
  if (runWorker) relayWorker.run(promClient);
  if (runRest) restListener.run();

  if (readinessPort) {
    const Net = require("net");
    const readinessServer = new Net.Server();
    readinessServer.listen(readinessPort, function () {
      logger.info("listening for readiness requests on port " + readinessPort);
    });

    readinessServer.on("connection", function (socket: any) {
      //logger.debug("readiness connection");
    });
  }
} else {
  logger.error("Initialization failed.");
}
