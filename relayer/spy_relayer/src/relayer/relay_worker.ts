import { hexToUint8Array, parseTransferPayload } from "@certusone/wormhole-sdk";
import { importCoreWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import { getRelayerEnvironment, RelayerEnvironment } from "../configureEnv";
import { getLogger, getScopedLogger, ScopedLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";
import {
  clearRedis,
  connectToRedis,
  demoteWorkingRedis,
  monitorRedis,
  RedisTables,
  resetPayload,
  Status,
  StorePayload,
  storePayloadFromJson,
  storePayloadToJson,
  WorkerInfo,
} from "../helpers/redisHelper";
import { sleep } from "../helpers/utils";
import { relay } from "./relay";
import { getBackend } from "../backends";

const WORKER_THREAD_RESTART_MS = 10 * 1000;
const AUDITOR_THREAD_RESTART_MS = 10 * 1000;
const AUDIT_INTERVAL_MS = 30 * 1000;
const WORKER_INTERVAL_MS = 5 * 1000;
const REDIS_RETRY_MS = 10 * 1000;

let metrics: PromHelper;

const logger = getLogger();
let relayerEnv: RelayerEnvironment;

type WorkableItem = {
  key: string;
  value: string;
};

export function init(): boolean {
  try {
    relayerEnv = getRelayerEnvironment();
  } catch (e) {
    logger.error(
      "Encountered error while initiating the relayer environment: " + e
    );
    return false;
  }

  return true;
}

/** Initialize metrics for each chain and the worker infos */
function createWorkerInfos(metrics: PromHelper): WorkerInfo[] {
  let workerArray: WorkerInfo[] = new Array();
  let index = 0;
  relayerEnv.supportedChains.forEach((chain) => {
    // initialize per chain metrics
    metrics.incSuccesses(chain.chainId, 0);
    metrics.incConfirmed(chain.chainId, 0);
    metrics.incFailures(chain.chainId, 0);
    metrics.incRollback(chain.chainId, 0);
    chain.walletPrivateKey?.forEach((key) => {
      workerArray.push({
        walletPrivateKey: key,
        index: index,
        targetChainId: chain.chainId,
        targetChainName: chain.chainName,
      });
      index++;
    });
    // TODO: Name the solanaprivatekey property the same as the non-solana one
    chain.solanaPrivateKey?.forEach((key) => {
      workerArray.push({
        walletPrivateKey: key,
        index: index,
        targetChainId: chain.chainId,
        targetChainName: chain.chainName,
      });
      index++;
    });
  });
  logger.info("will use " + workerArray.length + " workers");
  return workerArray;
}

/** Spawn relay worker and auditor threads for all chains */
async function spawnWorkerThreads(workerArray: WorkerInfo[]) {
  workerArray.forEach((workerInfo) => {
    spawnWorkerThread(workerInfo);
    spawnAuditorThread(workerInfo);
  });
}

async function spawnAuditorThread(workerInfo: WorkerInfo) {
  logger.info(
    `Spinning up auditor thread for target chain [${workerInfo.targetChainName}-${workerInfo.index}]`
  );

  //At present, due to the try catch inside the while loop, this thread should never crash.
  const auditorPromise = getBackend()
    .relayer.runAuditor(workerInfo)
    .catch(async (error: Error) => {
      logger.error(
        `Fatal crash on auditor thread ${workerInfo.targetChainName}-${workerInfo.index}`
      );
      logger.error("error message: " + error.message);
      logger.error("error trace: " + error.stack);
      await sleep(AUDITOR_THREAD_RESTART_MS);
      spawnAuditorThread(workerInfo);
    });

  return auditorPromise;
}

export async function run(ph: PromHelper) {
  metrics = ph;

  if (relayerEnv.clearRedisOnInit) {
    logger.info("Clearing REDIS as per tunable...");
    await clearRedis();
  } else if (relayerEnv.demoteWorkingOnInit) {
    logger.info("Demoting Working to Incoming as per tunable...");
    await demoteWorkingRedis();
  } else {
    logger.info("NOT clearing REDIS.");
  }

  let workerArray: WorkerInfo[] = createWorkerInfos(metrics);

  spawnWorkerThreads(workerArray);
  try {
    monitorRedis(metrics);
  } catch (e) {
    logger.error("Failed to kick off monitorRedis: " + e);
  }
}

// Redis does not guarantee ordering.  Therefore, it is possible that if workItems are
// pulled out one at a time, then some workItems could stay in the table indefinitely.
// This function gathers all the items available at this moment to work on.
async function findWorkableItems(
  workerInfo: WorkerInfo,
  relayLogger: ScopedLogger
): Promise<WorkableItem[]> {
  const logger = getScopedLogger(["findWorkableItems"], relayLogger);
  try {
    let workableItems: WorkableItem[] = [];
    const redisClient = await connectToRedis();
    if (!redisClient) {
      logger.error("Failed to connect to redis inside findWorkableItems()!");
      return workableItems;
    }
    await redisClient.select(RedisTables.INCOMING);
    for await (const si_key of redisClient.scanIterator()) {
      const si_value = await redisClient.get(si_key);
      if (si_value) {
        let storePayload: StorePayload = storePayloadFromJson(si_value);
        // Check to see if this worker should handle this VAA
        if (workerInfo.targetChainId !== 0) {
          const { parse_vaa } = await importCoreWasm();
          const parsedVAA = parse_vaa(hexToUint8Array(storePayload.vaa_bytes));
          const payloadBuffer: Buffer = Buffer.from(parsedVAA.payload);
          const transferPayload = parseTransferPayload(payloadBuffer);
          const tgtChainId = transferPayload.targetChain;
          if (tgtChainId !== workerInfo.targetChainId) {
            // Skipping mismatched chainId
            continue;
          }
        }

        // Check to see if this is a retry and if it is time to retry
        if (storePayload.retries > 0) {
          const BACKOFF_TIME = 1000; // 1 second in milliseconds
          const MAX_BACKOFF_TIME = 4 * 60 * 60 * 1000; // 4 hours in milliseconds
          // calculate retry time
          const now: Date = new Date();
          const old: Date = new Date(storePayload.timestamp);
          const timeDelta: number = now.getTime() - old.getTime(); // delta is in mS
          const waitTime: number = Math.min(
            BACKOFF_TIME * 10 ** storePayload.retries, //First retry is 10 second, then 100, 1,000... Max of 4 hours.
            MAX_BACKOFF_TIME
          );
          if (timeDelta < waitTime) {
            // Not enough time has passed
            continue;
          }
        }
        workableItems.push({ key: si_key, value: si_value });
      }
    }
    redisClient.quit();
    return workableItems;
  } catch (e: any) {
    logger.error(
      "Recoverable exception scanning REDIS for workable items: " + e.message
    );
    logger.error(e);
    return [];
  }
}

/** Spin up one worker for each (chainId, privateKey) combo. */
async function spawnWorkerThread(workerInfo: WorkerInfo) {
  logger.info(
    "Spinning up worker[" +
      workerInfo.index +
      "] to handle target chain " +
      workerInfo.targetChainId +
      ` / ${workerInfo.targetChainName}`
  );

  const workerPromise = doWorkerThread(workerInfo).catch(async (error) => {
    logger.error(
      "Fatal crash on worker thread: index " +
        workerInfo.index +
        " chainId " +
        workerInfo.targetChainId
    );
    logger.error("error message: " + error.message);
    logger.error("error trace: " + error.stack);
    await sleep(WORKER_THREAD_RESTART_MS);
    spawnWorkerThread(workerInfo);
  });

  return workerPromise;
}

async function doWorkerThread(workerInfo: WorkerInfo) {
  // relay-worker-solana-1
  const loggerName = `relay-worker-${workerInfo.targetChainName}-${workerInfo.index}`;
  const relayLogger = getScopedLogger([loggerName]);
  const backend = getBackend().relayer;
  while (true) {
    // relayLogger.debug("Finding workable items.");
    const workableItems: WorkableItem[] = await findWorkableItems(
      workerInfo,
      relayLogger
    );
    // relayLogger.debug("Found items: %o", workableItems);
    let i: number = 0;
    for (i = 0; i < workableItems.length; i++) {
      const workItem: WorkableItem = workableItems[i];
      if (workItem) {
        //This will attempt to move the workable item to the WORKING table
        relayLogger.debug("Moving item: %o", workItem);
        if (await moveToWorking(workItem, relayLogger)) {
          relayLogger.info("Moved key to WORKING table: %s", workItem.key);
          await backend.process(
            workItem.key,
            workerInfo.walletPrivateKey,
            relayLogger
          );
        } else {
          relayLogger.error(
            "Cannot move work item from INCOMING to WORKING: %s",
            workItem.key
          );
        }
      }
    }
    // relayLogger.debug(
    //   "Taking a break for %i seconds",
    //   WORKER_INTERVAL_MS / 1000
    // );
    await sleep(WORKER_INTERVAL_MS);
  }
}

async function moveToWorking(
  workItem: WorkableItem,
  relayLogger: ScopedLogger
): Promise<boolean> {
  const logger = getScopedLogger(["moveToWorking"], relayLogger);
  try {
    const redisClient = await connectToRedis();
    if (!redisClient) {
      logger.error("Failed to connect to Redis.");
      return false;
    }
    // Move this entry from incoming store to working store
    await redisClient.select(RedisTables.INCOMING);
    if ((await redisClient.del(workItem.key)) === 0) {
      logger.info("The key %s no longer exists in INCOMING", workItem.key);
      await redisClient.quit();
      return false;
    }
    await redisClient.select(RedisTables.WORKING);
    // If this VAA is already in the working store, then no need to add it again.
    // This handles the case of duplicate VAAs from multiple guardians
    const checkVal = await redisClient.get(workItem.key);
    if (!checkVal) {
      let payload: StorePayload = storePayloadFromJson(workItem.value);
      payload.status = Status.Pending;
      await redisClient.set(workItem.key, storePayloadToJson(payload));
      await redisClient.quit();
      return true;
    } else {
      metrics.incAlreadyExec();
      logger.debug("Dropping request %s as already processed", workItem.key);
      await redisClient.quit();
      return false;
    }
  } catch (e: any) {
    logger.error("Recoverable exception moving item to working: " + e.message);
    logger.error("%s => %s", workItem.key, workItem.value);
    logger.error(e);
    return false;
  }
}
