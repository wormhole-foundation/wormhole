// import {
//   createSpyRPCServiceClient,
//   subscribeSignedVAA,
// } from "@certusone/wormhole-spydk";

// import {
//   ChainId,
//   CHAIN_ID_SOLANA,
//   CHAIN_ID_TERRA,
//   hexToUint8Array,
//   uint8ArrayToHex,
//   parseTransferPayload,
//   getEmitterAddressEth,
//   getEmitterAddressSolana,
//   getEmitterAddressTerra,
// } from "@certusone/wormhole-sdk";

// import { storeKeyFromParsedVAA, storePayloadFromVaaBytes } from "./helpers";
import { hexToUint8Array, parseTransferPayload } from "@certusone/wormhole-sdk";
import { importCoreWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import { getRelayerEnvironment, RelayerEnvironment } from "../configureEnv";
import { getLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";
import {
  clearRedis,
  connectToRedis,
  RedisTables,
  RelayResult,
  Status,
  StorePayload,
  storePayloadFromJson,
  storePayloadToJson,
  WorkerInfo,
} from "../helpers/redisHelper";
import { sleep } from "../helpers/utils";
import { relay } from "./relay";
import { collectWallets } from "./walletMonitor";

const WORKER_THREAD_RESTART_MS = 10 * 1000;
const AUDITOR_THREAD_RESTART_MS = 10 * 1000;
const AUDIT_INTERVAL_MS = 30 * 1000;
const REDIS_RETRY_MS = 10 * 1000;

let metrics: PromHelper;

const logger = getLogger();
let relayerEnv: RelayerEnvironment;

type WorkableItem = {
  key: string;
  value: string;
};

export function init(runWorker: boolean): boolean {
  if (!runWorker) return true;

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

function createWorkerInfos() {
  let workerArray: WorkerInfo[] = new Array();
  let index = 0;
  relayerEnv.supportedChains.forEach((chain) => {
    chain.walletPrivateKey?.forEach((key) => {
      workerArray.push({
        walletPrivateKey: key,
        index: index,
        targetChainId: chain.chainId,
      });
      index++;
    });
    chain.solanaPrivateKey?.forEach((key) => {
      workerArray.push({
        walletPrivateKey: key,
        index: index,
        targetChainId: chain.chainId,
      });
      index++;
    });
  });
  logger.info("will use " + workerArray.length + " workers");
  return workerArray;
}

async function spawnWorkerThreads(workerArray: WorkerInfo[]) {
  workerArray.forEach((workerInfo) => {
    spawnWorkerThread(workerInfo);
    spawnAuditorThread(workerInfo);
  });
}

async function spawnAuditorThread(workerInfo: WorkerInfo) {
  logger.info(
    "Spinning up auditor thread[" +
      workerInfo.index +
      "] to handle targetChainId " +
      workerInfo.targetChainId
  );

  //At present, due to the try catch inside the while loop, this thread should never crash.
  const auditorPromise = doAuditorThread(workerInfo).catch(
    async (error: Error) => {
      logger.error(
        "Fatal crash on auditor thread: index " +
          workerInfo.index +
          " chainId " +
          workerInfo.targetChainId
      );
      logger.error("error message: " + error.message);
      logger.error("error trace: " + error.stack);
      await sleep(AUDITOR_THREAD_RESTART_MS);
      spawnAuditorThread(workerInfo);
    }
  );

  return auditorPromise;
}

//One auditor thread should be spawned per worker. This is perhaps overkill, but auditors
//should not be allowed to block workers, or other auditors.
async function doAuditorThread(workerInfo: WorkerInfo) {
  while (true) {
    try {
      let redisClient: any = null;
      while (!redisClient) {
        redisClient = await connectToRedis();
        if (!redisClient) {
          logger.error("audit worker failed to connect to redis!");
          await sleep(REDIS_RETRY_MS);
        }
      }
      await redisClient.select(RedisTables.WORKING);
      let now: Date = new Date();
      for await (const si_key of redisClient.scanIterator()) {
        const si_value = await redisClient.get(si_key);
        if (!si_value) {
          continue;
        }

        let storePayload: StorePayload = storePayloadFromJson(si_value);
        try {
          const { parse_vaa } = await importCoreWasm();
          const parsedVAA = parse_vaa(hexToUint8Array(storePayload.vaa_bytes));
          const payloadBuffer: Buffer = Buffer.from(parsedVAA.payload);
          const transferPayload = parseTransferPayload(payloadBuffer);

          const chain = transferPayload.targetChain;
          if (chain !== workerInfo.targetChainId) {
            continue;
          }
        } catch (e) {
          logger.error("Audit worker Failed to parse a stored VAA: " + e);
          logger.error("si_value of failure: " + si_value);
          continue;
        }
        logger.debug(
          "audit thread: si_key " +
            si_key +
            " => status: " +
            storePayload.status +
            ", timestamp: " +
            storePayload.timestamp +
            ", retries: " +
            storePayload.retries
        );
        // Let things sit in here for 10 minutes
        // After that:
        //    - Toss totally failed VAAs
        //    - Check to see if successful transactions were rolled back
        //    - Put roll backs into INCOMING table
        //    - Toss legitimately completed transactions
        let old: Date = new Date(storePayload.timestamp);
        let timeDelta: number = now.getTime() - old.getTime(); // delta is in mS
        const TEN_MINUTES = 600000;
        logger.debug(
          "Audit worker checking timestamps:  now: " +
            now.toString() +
            ", old: " +
            old.toString() +
            ", delta: " +
            timeDelta
        );
        if (timeDelta > TEN_MINUTES) {
          // Deal with this item
          if (storePayload.status === Status.FatalError) {
            // Done with this failed transaction
            logger.debug("Audit thread: discarding FatalError.");
            await redisClient.del(si_key);
            continue;
          } else if (storePayload.status === Status.Completed) {
            // Check for rollback
            logger.debug("Audit thread: checking for rollback.");

            //TODO actually do an isTransferCompleted
            const rr = await relay(
              storePayload.vaa_bytes,
              true,
              workerInfo.walletPrivateKey
            );

            await redisClient.del(si_key);
            if (rr.status !== Status.Completed) {
              logger.info("Detected a rollback on " + si_key);
              // Remove this item from the WORKING table and move it to INCOMING
              await redisClient.select(RedisTables.INCOMING);
              await redisClient.set(si_key, si_value);
              await redisClient.select(RedisTables.WORKING);
            }
          } else if (storePayload.status === Status.Error) {
            logger.error("Audit thread received Error status.");
            continue;
          } else if (storePayload.status === Status.Pending) {
            logger.error("Audit thread received Pending status.");
            continue;
          } else {
            logger.error(
              "Audit thread: Unhandled Status of " + storePayload.status
            );
            console.log(
              "Audit thread: Unhandled Status of ",
              storePayload.status
            );
            continue;
          }
        }
      }
      redisClient.quit();
      // metrics.setDemoWalletBalance(now.getUTCSeconds());
      await sleep(AUDIT_INTERVAL_MS);
    } catch (e) {
      logger.error("spawnAuditorThread: caught exception: " + e);
    }
  }
}

export async function run(ph: PromHelper) {
  metrics = ph;

  if (relayerEnv.clearRedisOnInit) {
    logger.info("Clearing REDIS as per tunable...");
    await clearRedis();
  } else {
    logger.info("NOT clearing REDIS.");
  }

  let workerArray: WorkerInfo[] = createWorkerInfos();

  spawnWorkerThreads(workerArray);
  try {
    collectWallets(metrics);
  } catch (e) {
    logger.error("Failed to kick off collectWallets: " + e);
  }
}

async function processRequest(
  myWorkerIdx: number,
  key: string,
  myPrivateKey: any
) {
  try {
    logger.debug("[" + myWorkerIdx + "] Processing request [" + key + "]...");
    // Get the entry from the working store
    const rClient = await connectToRedis();
    if (!rClient) {
      logger.error(
        "[" + myWorkerIdx + "] failed to connect to Redis in processRequest"
      );
      return;
    }
    await rClient.select(RedisTables.WORKING);
    let value: string | null = await rClient.get(key);
    if (!value) {
      logger.error(
        "[" + myWorkerIdx + "] processRequest could not find key [" + key + "]"
      );
      return;
    }
    let payload: StorePayload = storePayloadFromJson(value);
    if (payload.status !== Status.Pending) {
      logger.info(
        "[" +
          myWorkerIdx +
          "] This key [" +
          key +
          "] has already been processed."
      );
      return;
    }
    // Actually do the processing here and update status and time field
    let relayResult: RelayResult;
    try {
      logger.info(
        "[" +
          myWorkerIdx +
          "] processRequest() - Calling with vaa_bytes [" +
          payload.vaa_bytes +
          "]"
      );
      relayResult = await relay(payload.vaa_bytes, false, myPrivateKey);
      logger.info(
        "[" + myWorkerIdx + "] processRequest() - relay returned: %o",
        relayResult.status
      );
    } catch (e: any) {
      logger.error(
        "[" +
          myWorkerIdx +
          "] processRequest() - failed to relay transfer vaa: %o",
        e
      );

      relayResult = {
        status: Status.Error,
        result: "Failure",
      };
      if (e && e.message) {
        relayResult.result = e.message;
      }
    }

    const MAX_RETRIES = 10;
    let retry: boolean = false;
    if (relayResult.status === Status.Completed) {
      metrics.incSuccesses();
    } else {
      metrics.incFailures();
      if (payload.retries >= MAX_RETRIES) {
        relayResult.status = Status.FatalError;
      }
      if (relayResult.status === Status.FatalError) {
        // Invoke fatal error logic here!
        payload.retries = MAX_RETRIES;
      } else {
        // Invoke retry logic here!
        retry = true;
      }
    }

    // Put result back into store
    payload.status = relayResult.status;
    payload.timestamp = new Date().toString();
    payload.retries++;
    value = storePayloadToJson(payload);
    if (!retry || payload.retries > MAX_RETRIES) {
      await rClient.set(key, value);
    } else {
      // Remove from the working table
      await rClient.del(key);
      // Put this back into the incoming table
      await rClient.select(RedisTables.INCOMING);
      await rClient.set(key, value);
    }
    await rClient.quit();
  } catch (e: any) {
    logger.error("Unexpected error in processRequest: " + e.message);
    logger.error("request key: " + key);
    logger.error("worker index: " + myWorkerIdx);
    logger.error("Exception trace: " + e.stack);
    return [];
  }
}

// Redis does not guarantee ordering.  Therefore, it is possible that if workItems are
// pulled out one at a time, then some workItems could stay in the table indefinitely.
// This function gathers all the items available at this moment to work on.
async function findWorkableItems(
  workerInfo: WorkerInfo
): Promise<WorkableItem[]> {
  try {
    let workableItems: WorkableItem[] = [];
    const redisClient = await connectToRedis();
    if (!redisClient) {
      logger.error(
        "Worker [" +
          workerInfo.index +
          "] Failed to connect to redis inside findWorkableItems()!"
      );
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
            // logger.debug(
            //   "Skipping mismatched chainId.  Received: " +
            //     tgtChainId +
            //     ", want: " +
            //     workerInfo.targetChainId
            // );
            continue;
          }
        }

        // Check to see if this is a retry and if it is time to retry
        if (storePayload.retries > 0) {
          const BACKOFF_TIME = 10000; // 10 seconds in milliseconds
          const MAX_BACKOFF_TIME = 86400000; // 24 hours in milliseconds
          // calculate retry time
          const now: Date = new Date();
          const old: Date = new Date(storePayload.timestamp);
          const timeDelta: number = now.getTime() - old.getTime(); // delta is in mS
          const waitTime: number = Math.min(
            BACKOFF_TIME ** storePayload.retries,
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
    logger.error("Exception trace: " + e.stack);
    return [];
  }
}

//One worker should be spawned for each chainId+privateKey combo.
async function spawnWorkerThread(workerInfo: WorkerInfo) {
  logger.info(
    "Spinning up worker[" +
      workerInfo.index +
      "] to handle targetChainId " +
      workerInfo.targetChainId
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
  while (true) {
    const workableItems: WorkableItem[] = await findWorkableItems(workerInfo);
    // logger.debug( "[" + workerInfo.index + "] received " + workableItems.length + " workable items.");
    let i: number = 0;
    for (i = 0; i < workableItems.length; i++) {
      const workItem: WorkableItem = workableItems[i];
      if (workItem) {
        // logger.debug("attempting to move key: " + workItem.key);
        //This will attempt to move the workable item to the WORKING table
        if (await moveToWorking(workerInfo, workItem)) {
          // logger.debug("Moved key: " + workItem.key + " to WORKING table.");
          await processRequest(
            workerInfo.index,
            workItem.key,
            workerInfo.walletPrivateKey
          );
          // logger.debug("Finished processing key: " + workItem.key);
        } else {
          logger.error("Cannot move work item from INCOMING to WORKING.");
        }
      }
    }
  }
}

async function moveToWorking(
  workerInfo: WorkerInfo,
  workItem: WorkableItem
): Promise<boolean> {
  try {
    const redisClient = await connectToRedis();
    if (!redisClient) {
      logger.error("moveToPending() - failed to connect to Redis.");
      return false;
    }
    // Move this entry from incoming store to working store
    await redisClient.select(RedisTables.INCOMING);
    if ((await redisClient.del(workItem.key)) === 0) {
      logger.info(
        "[" +
          workerInfo.index +
          "] The key [" +
          workItem.key +
          "] no longer exists in INCOMING"
      );
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
      logger.debug(
        "dropping request [" + workItem.key + "] as already processed"
      );
      await redisClient.quit();
      return false;
    }
  } catch (e: any) {
    logger.error("Recoverable exception moving item to working: " + e.message);
    logger.error("Problem item: " + workItem.key);
    logger.error("Problem value: " + workItem.value);
    logger.error("Exception trace: " + e.stack);
    return false;
  }
}
