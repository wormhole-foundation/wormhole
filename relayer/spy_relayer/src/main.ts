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
import * as walletMonitor from "./monitor";

const ARG_LISTEN_ONLY = "--listen_only";
const ARG_RELAY_ONLY = "--relay_only";
const ARG_WALLET_MONITOR_ONLY = "--wallet_monitor_only";
const ONLY_ONE_ARG_ERROR_MSG = `May only specify one of ${ARG_LISTEN_ONLY}, ${ARG_RELAY_ONLY}, or ${ARG_WALLET_MONITOR_ONLY}`;
const ONLY_ONE_ARG_ERROR_RESULT = `Multiple args found of ${ARG_LISTEN_ONLY}, ${ARG_RELAY_ONLY}, ${ARG_WALLET_MONITOR_ONLY}`;

setDefaultWasm("node");
const logger = getLogger();

// Load the relay config data.
let runListen: boolean = true;
let runRelayWorker: boolean = true;
let runRest: boolean = true;
let runWalletMonitor: boolean = true;
let foundOne: boolean = false;
let error: string = "";

for (let idx = 0; idx < process.argv.length; ++idx) {
  if (process.argv[idx] === ARG_LISTEN_ONLY) {
    if (foundOne) {
      logger.error(ONLY_ONE_ARG_ERROR_MSG);
      error = ONLY_ONE_ARG_ERROR_RESULT;
      break;
    }

    logger.info("spy_relay is running in listen only mode");
    runRelayWorker = false;
    runWalletMonitor = false;
    foundOne = true;
  }

  if (process.argv[idx] === ARG_RELAY_ONLY) {
    if (foundOne) {
      logger.error(ONLY_ONE_ARG_ERROR_MSG);
      error = ONLY_ONE_ARG_ERROR_RESULT;
      break;
    }

    logger.info("spy_relay is running in relay only mode");
    runListen = false;
    runRest = false;
    runWalletMonitor = false;
    foundOne = true;
  }

  if (process.argv[idx] === ARG_WALLET_MONITOR_ONLY) {
    if (foundOne) {
      logger.error(ONLY_ONE_ARG_ERROR_MSG);
      error = ONLY_ONE_ARG_ERROR_RESULT;
      break;
    }

    logger.info("spy_relay is running in wallet monitor only mode");
    runListen = false;
    runRest = false;
    runRelayWorker = false;
    foundOne = true;
  }
}

if (!foundOne) {
  logger.info("spy_relay is running both the listener and relayer");
}

const runAll: boolean = runListen && runRelayWorker && runWalletMonitor;
if (runListen && !spyListener.init()) {
  process.exit(1);
}

if (runRelayWorker && !relayWorker.init()) {
  process.exit(1);
}

if (runRest && !restListener.init()) {
  process.exit(1);
}

if (runWalletMonitor && !walletMonitor.init()) {
  process.exit(1);
}

if (error) {
  logger.error(error);
  process.exit(1);
}

const commonEnv = getCommonEnvironment();
const { promPort, readinessPort } = commonEnv;
logger.info("prometheus client listening on port " + promPort);
let promClient: PromHelper;
if (runAll) {
  promClient = new PromHelper("spy_relay", promPort, PromMode.All);
} else if (runListen) {
  promClient = new PromHelper("spy_relay", promPort, PromMode.Listen);
} else if (runRelayWorker) {
  promClient = new PromHelper("spy_relay", promPort, PromMode.Relay);
} else if (runWalletMonitor) {
  promClient = new PromHelper("spy_relay", promPort, PromMode.WalletMonitor);
} else {
  logger.error("Invalid run mode for Prometheus");
  promClient = new PromHelper("spy_relay", promPort, PromMode.All);
}

redisHelper.init(promClient);

if (runListen) spyListener.run(promClient);
if (runRelayWorker) relayWorker.run(promClient);
if (runRest) restListener.run();
if (runWalletMonitor) walletMonitor.run(promClient);

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
