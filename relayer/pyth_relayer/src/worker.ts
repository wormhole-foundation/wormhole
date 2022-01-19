import { Mutex } from "async-mutex";
let CondVar = require("condition-variable");

import { setDefaultWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import { uint8ArrayToHex } from "@certusone/wormhole-sdk";

import * as helpers from "./helpers";
import { logger } from "./helpers";
import * as main from "./relay/main";
import { PromHelper } from "./promHelpers";

const mutex = new Mutex();
let condition = new CondVar();
let conditionTimeout = 20000;

type PendingPayload = {
  vaa_bytes: string;
  pa: helpers.PythPriceAttestation;
  receiveTime: Date;
  seqNum: number;
};

let pendingMap = new Map<string, PendingPayload>(); // The key to this is price_id. Note that Map maintains insertion order, not key order.

type ProductData = {
  key: string;
  lastTimePublished: Date;
  numTimesPublished: number;
  lastPa: helpers.PythPriceAttestation;
  lastResult: any;
};

type CurrentEntry = {
  pendingEntry: PendingPayload;
  currObj: ProductData;
};

let productMap = new Map<string, ProductData>(); // The key to this is price_id

let connectionData: main.ConnectionData;
let metrics: PromHelper;
let nextBalanceQueryTimeAsMs: number = 0;
let balanceQueryInterval = 0;
let walletTimeStamp: Date;
let maxPerBatch: number = 1;
let maxAttempts: number = 2;
let retryDelayInMs: number = 0;

export function init(runWorker: boolean): boolean {
  if (!runWorker) return true;

  try {
    connectionData = main.connectRelayer();
  } catch (e) {
    logger.error("failed to load connection config: %o", e);
    return false;
  }

  if (process.env.MAX_MSGS_PER_BATCH) {
    maxPerBatch = parseInt(process.env.MAX_MSGS_PER_BATCH);
  }

  if (maxPerBatch <= 0) {
    logger.error(
      "Environment variable MAX_MSGS_PER_BATCH has an invalid value of " +
        maxPerBatch +
        ", must be greater than zero."
    );

    return false;
  }

  if (process.env.RETRY_MAX_ATTEMPTS) {
    maxAttempts = parseInt(process.env.RETRY_MAX_ATTEMPTS);
  }

  if (maxAttempts <= 0) {
    logger.error(
      "Environment variable RETRY_MAX_ATTEMPTS has an invalid value of " +
        maxAttempts +
        ", must be greater than zero."
    );

    return false;
  }

  if (process.env.RETRY_DELAY_IN_MS) {
    retryDelayInMs = parseInt(process.env.RETRY_DELAY_IN_MS);
  }

  if (retryDelayInMs < 0) {
    logger.error(
      "Environment variable RETRY_DELAY_IN_MS has an invalid value of " +
        retryDelayInMs +
        ", must be positive or zero."
    );

    return false;
  }

  return true;
}

export async function run(met: PromHelper) {
  setDefaultWasm("node");

  metrics = met;

  await mutex.runExclusive(async () => {
    logger.info(
      "will attempt to relay each pyth message at most " +
        maxAttempts +
        " times, with a delay of " +
        retryDelayInMs +
        " milliseconds between attempts, will batch up to " +
        maxPerBatch +
        " pyth messages in a batch"
    );

    if (process.env.BAL_QUERY_INTERVAL) {
      balanceQueryInterval = parseInt(process.env.BAL_QUERY_INTERVAL);
    }

    await main.setAccountNum(connectionData);
    logger.info(
      "wallet account number is " + connectionData.terraData.walletAccountNum
    );

    await main.setSeqNum(connectionData);
    logger.info(
      "initial wallet sequence number is " +
        connectionData.terraData.walletSeqNum
    );

    let balance = await main.queryBalance(connectionData);
    if (!isNaN(balance)) {
      walletTimeStamp = new Date();
    }
    if (balanceQueryInterval !== 0) {
      logger.info(
        "initial wallet balance is " +
          balance +
          ", will query every " +
          balanceQueryInterval +
          " milliseconds."
      );
      metrics.setWalletBalance(balance);

      nextBalanceQueryTimeAsMs = new Date().getTime() + balanceQueryInterval;
    } else {
      logger.info("initial wallet balance is " + balance);
      metrics.setWalletBalance(balance);
    }

    await condition.wait(computeTimeout(), callBack);
  });
}

async function callBack(err: any, result: any) {
  logger.debug(
    "entering callback, pendingEvents: " +
      pendingMap.size +
      ", err: %o, result: %o",
    err,
    result
  );
  // condition = null;
  // await helpers.sleep(10000);
  // logger.debug("done with long sleep");
  let done = false;
  do {
    let currObjs = new Array<CurrentEntry>();
    let messages = new Array<string>();

    await mutex.runExclusive(async () => {
      condition = null;
      logger.debug("in callback, getting pending events.");
      await getPendingEventsAlreadyLocked(currObjs, messages);

      if (currObjs.length === 0) {
        done = true;
        condition = new CondVar();
        await condition.wait(computeTimeout(), callBack);
      }
    });

    if (currObjs.length !== 0) {
      logger.debug("in callback, relaying " + currObjs.length + " events.");
      let sendTime = new Date();
      let retVal: number;
      let relayResult: any;
      [retVal, relayResult] = await relayEventsNotLocked(messages);

      await mutex.runExclusive(async () => {
        logger.debug("in callback, finalizing " + currObjs.length + " events.");
        await finalizeEventsAlreadyLocked(
          currObjs,
          retVal,
          relayResult,
          sendTime
        );

        if (pendingMap.size === 0) {
          logger.debug("in callback, rearming the condition.");
          done = true;
          condition = new CondVar();
          await condition.wait(computeTimeout(), callBack);
        }
      });
    }
  } while (!done);

  logger.debug("leaving callback.");
}

function computeTimeout(): number {
  if (balanceQueryInterval !== 0) {
    let now = new Date().getTime();
    if (now < nextBalanceQueryTimeAsMs) {
      return nextBalanceQueryTimeAsMs - now;
    }

    return 0;
  }

  return conditionTimeout;
}

async function getPendingEventsAlreadyLocked(
  currObjs: Array<CurrentEntry>,
  messages: Array<string>
) {
  while (pendingMap.size !== 0 && currObjs.length < maxPerBatch) {
    const first = pendingMap.entries().next();
    logger.debug("processing event with key [" + first.value[0] + "]");
    const pendingValue = first.value[1];
    let pendingKey = pendingValue.pa.priceId;
    let currObj = productMap.get(pendingKey);
    if (currObj) {
      currObj.lastPa = pendingValue.pa;
      currObj.lastTimePublished = new Date();
      productMap.set(pendingKey, currObj);
      logger.debug(
        "processing update " +
          currObj.numTimesPublished +
          " for [" +
          pendingKey +
          "], seq num " +
          pendingValue.seqNum
      );
    } else {
      logger.debug(
        "processing first update for [" +
          pendingKey +
          "], seq num " +
          pendingValue.seqNum
      );
      currObj = {
        key: pendingKey,
        lastPa: pendingValue.pa,
        lastTimePublished: new Date(),
        numTimesPublished: 0,
        lastResult: "",
      };
      productMap.set(pendingKey, currObj);
    }

    currObjs.push({ pendingEntry: pendingValue, currObj: currObj });
    messages.push(pendingValue.vaa_bytes);
    pendingMap.delete(first.value[0]);
  }

  if (currObjs.length !== 0) {
    for (let idx = 0; idx < currObjs.length; ++idx) {
      pendingMap.delete(currObjs[idx].currObj.key);
    }
  }
}

const RELAY_SUCCESS: number = 0;
const RELAY_FAIL: number = 1;
const RELAY_ALREADY_EXECUTED: number = 2;
const RELAY_TIMEOUT: number = 3;
const RELAY_SEQ_NUM_MISMATCH: number = 4;
const RELAY_INSUFFICIENT_FUNDS: number = 5;

async function relayEventsNotLocked(
  messages: Array<string>
): Promise<[number, any]> {
  let retVal: number = RELAY_SUCCESS;
  let relayResult: any;
  let retry: boolean = false;

  for (let attempt = 0; attempt < maxAttempts; ++attempt) {
    retVal = RELAY_SUCCESS;
    retry = false;

    try {
      relayResult = await main.relay(messages, connectionData);
      if (relayResult.txhash) {
        if (
          relayResult.raw_log &&
          relayResult.raw_log.search("VaaAlreadyExecuted") >= 0
        ) {
          relayResult = "Already Executed: " + relayResult.txhash;
          retVal = RELAY_ALREADY_EXECUTED;
        } else if (
          relayResult.raw_log &&
          relayResult.raw_log.search("insufficient funds") >= 0
        ) {
          logger.error(
            "relay failed due to insufficient funds: %o",
            relayResult
          );
          connectionData.terraData.walletSeqNum =
            connectionData.terraData.walletSeqNum - 1;
          retVal = RELAY_INSUFFICIENT_FUNDS;
        } else if (
          relayResult.raw_log &&
          relayResult.raw_log.search("failed") >= 0
        ) {
          logger.error("relay seems to have failed: %o", relayResult);
          retVal = RELAY_FAIL;
          retry = true;
        } else {
          relayResult = relayResult.txhash;
        }
      } else {
        retVal = RELAY_FAIL;
        retry = true;
        if (relayResult.message) {
          relayResult = relayResult.message;
        } else {
          logger.error("No txhash: %o", relayResult);
          relayResult = "No txhash";
        }
      }
    } catch (e: any) {
      if (
        e.message &&
        e.message.search("timeout") >= 0 &&
        e.message.search("exceeded") >= 0
      ) {
        logger.error("relay timed out: %o", e);
        retVal = RELAY_TIMEOUT;
        retry = true;
      } else {
        logger.error("relay failed: %o", e);
        if (e.response && e.response.data) {
          if (
            e.response.data.error &&
            e.response.data.error.search("VaaAlreadyExecuted") >= 0
          ) {
            relayResult = "Already Executed";
            retVal = RELAY_ALREADY_EXECUTED;
          } else if (
            e.response.data.message &&
            e.response.data.message.search("account sequence mismatch") >= 0
          ) {
            relayResult = e.response.data.message;
            retVal = RELAY_SEQ_NUM_MISMATCH;
            retry = true;

            logger.debug(
              "wallet sequence number is out of sync, querying the current value"
            );
            await main.setSeqNum(connectionData);
            logger.info(
              "wallet seq number is now " +
                connectionData.terraData.walletSeqNum
            );
          } else {
            retVal = RELAY_FAIL;
            retry = true;
            if (e.message) {
              relayResult = "Error: " + e.message;
            } else {
              relayResult = "Error: unexpected exception";
            }
          }
        } else {
          retVal = RELAY_FAIL;
          retry = true;
          if (e.message) {
            relayResult = "Error: " + e.message;
          } else {
            relayResult = "Error: unexpected exception";
          }
        }
      }
    }

    logger.debug(
      "relay attempt complete: retVal: " +
        retVal +
        ", retry: " +
        retry +
        ", attempt " +
        attempt +
        " of " +
        maxAttempts
    );

    if (!retry) {
      break;
    } else {
      metrics.incRetries();
      if (retryDelayInMs != 0) {
        logger.debug(
          "delaying for " + retryDelayInMs + " milliseconds before retrying"
        );
        await helpers.sleep(retryDelayInMs * (attempt + 1));
      }
    }
  }

  if (retry) {
    logger.error("failed to relay batch, retry count exceeded!");
    metrics.incRetriesExceeded();
  }

  return [retVal, relayResult];
}

async function finalizeEventsAlreadyLocked(
  currObjs: Array<CurrentEntry>,
  retVal: number,
  relayResult: any,
  sendTime: Date
) {
  for (let idx = 0; idx < currObjs.length; ++idx) {
    let currObj = currObjs[idx].currObj;
    let currEntry = currObjs[idx].pendingEntry;
    currObj.lastResult = relayResult;
    currObj.numTimesPublished = currObj.numTimesPublished + 1;
    if (retVal == RELAY_SUCCESS) {
      metrics.incSuccesses();
    } else if (retVal == RELAY_ALREADY_EXECUTED) {
      metrics.incAlreadyExec();
    } else if (retVal == RELAY_TIMEOUT) {
      metrics.incTransferTimeout();
      metrics.incFailures();
    } else if (retVal == RELAY_SEQ_NUM_MISMATCH) {
      metrics.incSeqNumMismatch();
      metrics.incFailures();
    } else if (retVal == RELAY_INSUFFICIENT_FUNDS) {
      metrics.incInsufficentFunds();
      metrics.incFailures();
    } else {
      metrics.incFailures();
    }
    productMap.set(currObj.key, currObj);

    let completeTime = new Date();
    metrics.setSeqNum(currEntry.seqNum);
    metrics.addCompleteTime(
      completeTime.getTime() - currEntry.receiveTime.getTime()
    );

    logger.info(
      "complete: priceId: " +
        currEntry.pa.priceId +
        ", seqNum: " +
        currEntry.seqNum +
        ", price: " +
        helpers.computePrice(currEntry.pa.price, currEntry.pa.exponent) +
        ", ci: " +
        helpers.computePrice(
          currEntry.pa.confidenceInterval,
          currEntry.pa.exponent
        ) +
        ", rcv2SendBegin: " +
        (sendTime.getTime() - currEntry.receiveTime.getTime()) +
        ", rcv2SendComplete: " +
        (completeTime.getTime() - currEntry.receiveTime.getTime()) +
        ", totalSends: " +
        currObj.numTimesPublished +
        ", result: " +
        relayResult
    );
  }

  let now = new Date();
  if (balanceQueryInterval > 0 && now.getTime() >= nextBalanceQueryTimeAsMs) {
    let balance = await main.queryBalance(connectionData);
    if (isNaN(balance)) {
      logger.error("failed to query wallet balance!");
    } else {
      if (!isNaN(balance)) {
        walletTimeStamp = new Date();
      }
      logger.info(
        "wallet balance: " +
          balance +
          ", update time: " +
          walletTimeStamp.toISOString()
      );
      metrics.setWalletBalance(balance);
    }
    nextBalanceQueryTimeAsMs = now.getTime() + balanceQueryInterval;
  }
}

export async function postEvent(
  vaaBytes: any,
  pa: helpers.PythPriceAttestation,
  sequence: number,
  receiveTime: Date
) {
  let event: PendingPayload = {
    vaa_bytes: uint8ArrayToHex(vaaBytes),
    pa: pa,
    receiveTime: receiveTime,
    seqNum: sequence,
  };
  let pendingKey = pa.priceId;
  // pendingKey = pendingKey + ":" + sequence;
  await mutex.runExclusive(() => {
    logger.debug("posting event with key [" + pendingKey + "]");
    pendingMap.set(pendingKey, event);
    if (condition) {
      logger.debug("hitting condition variable.");
      condition.complete(true);
    }
  });
}

export async function getStatus() {
  let result = "[";
  await mutex.runExclusive(() => {
    let first: boolean = true;
    for (let [key, value] of productMap) {
      if (first) {
        first = false;
      } else {
        result = result + ", ";
      }

      let item: object = {
        product_id: value.lastPa.productId,
        price_id: value.lastPa.priceId,
        price: helpers.computePrice(value.lastPa.price, value.lastPa.exponent),
        ci: helpers.computePrice(
          value.lastPa.confidenceInterval,
          value.lastPa.exponent
        ),
        num_times_published: value.numTimesPublished,
        last_time_published: value.lastTimePublished.toISOString(),
        result: value.lastResult,
      };

      result = result + JSON.stringify(item);
    }
  });

  result = result + "]";
  return result;
}

// Note that querying the contract does not update the sequence number, so we don't need to be locked.
export async function getPriceData(
  productId: string,
  priceId: string
): Promise<any> {
  let result: any;
  // await mutex.runExclusive(async () => {
  result = await main.query(productId, priceId);
  // });

  return result;
}
