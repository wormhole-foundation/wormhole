import {
  ChainId,
  hexToUint8Array,
  importCoreWasm,
  parseTransferPayload,
  uint8ArrayToHex,
} from "@certusone/wormhole-sdk";
import { Mutex } from "async-mutex";
import { createClient, RedisClientType } from "redis";
import { getCommonEnvironment } from "../configureEnv";
import { ParsedTransferPayload, ParsedVaa } from "../listener/validation";
import { chainIDStrings } from "../utils/wormhole";
import { getScopedLogger } from "./logHelper";
import { PromHelper } from "./promHelpers";
import { sleep } from "./utils";

const logger = getScopedLogger(["redisHelper"]);
const commonEnv = getCommonEnvironment();
const { redisHost, redisPort } = commonEnv;
let promHelper: PromHelper;

//Module internals
const redisMutex = new Mutex();
let redisQueue = new Array<[string, string]>();

export function getBackupQueue() {
  return redisQueue;
}

export enum RedisTables {
  INCOMING = 0,
  WORKING = 1,
}

export function init(ph: PromHelper): boolean {
  logger.info("will connect to redis at [" + redisHost + ":" + redisPort + "]");
  promHelper = ph;
  return true;
}

export async function connectToRedis() {
  let rClient;
  try {
    rClient = createClient({
      socket: {
        host: redisHost,
        port: redisPort,
      },
    });

    rClient.on("connect", function (err) {
      if (err) {
        logger.error(
          "connectToRedis: failed to connect to host [" +
            redisHost +
            "], port [" +
            redisPort +
            "]: %o",
          err
        );
      }
    });

    await rClient.connect();
  } catch (e) {
    logger.error(
      "connectToRedis: failed to connect to host [" +
        redisHost +
        "], port [" +
        redisPort +
        "]: %o",
      e
    );
  }

  return rClient;
}

export async function storeInRedis(name: string, value: string) {
  if (!name) {
    logger.error("storeInRedis: missing name");
    return;
  }
  if (!value) {
    logger.error("storeInRedis: missing value");
    return;
  }

  await redisMutex.runExclusive(async () => {
    logger.debug("storeInRedis: connecting to redis.");
    let redisClient;
    try {
      redisQueue.push([name, value]);
      redisClient = await connectToRedis();
      if (!redisClient) {
        logger.error(
          "Failed to connect to redis, enqueued vaa, there are now " +
            redisQueue.length +
            " enqueued events"
        );
        return;
      }

      logger.debug(
        "now connected to redis, attempting to push " +
          redisQueue.length +
          " queued items"
      );
      for (let item = redisQueue.pop(); item; item = redisQueue.pop()) {
        await addToRedis(redisClient, item[0], item[1]);
      }
    } catch (e) {
      logger.error(
        "Failed during redis item push. Currently" +
          redisQueue.length +
          " enqueued items"
      );
      logger.error(
        "encountered an exception while pushing items to redis %o",
        e
      );
    }

    try {
      if (redisClient) {
        await redisClient.quit();
      }
    } catch (e) {
      logger.error("Failed to quit redis client");
    }
  });

  promHelper.handleListenerMemqueue(redisQueue.length);
}

export async function addToRedis(
  redisClient: any,
  name: string,
  value: string
) {
  try {
    logger.debug("storeInRedis: storing in redis. name: " + name);
    await redisClient.select(RedisTables.INCOMING);
    await redisClient.set(name, value);

    logger.debug("storeInRedis: finished storing in redis.");
  } catch (e) {
    logger.error(
      "storeInRedis: failed to store to host [" +
        redisHost +
        "], port [" +
        redisPort +
        "]: %o",
      e
    );
  }
}

/** Redis key name for storing VAAs in the memory queue */
export function getKey(chainId: ChainId, address: string) {
  return chainId + ":" + address;
}

export enum Status {
  Pending = 1,
  Completed = 2,
  Error = 3,
  FatalError = 4,
}

export type RelayResult = {
  status: Status;
  result: string | null;
};

export type WorkerInfo = {
  index: number;
  targetChainId: number;
  walletPrivateKey: any;
};

export type StoreKey = {
  chain_id: number;
  emitter_address: string;
  sequence: number;
};

export type StorePayload = {
  vaa_bytes: string;
  status: Status;
  timestamp: string;
  retries: number;
};

/** Default redis payload */
export function initPayload(): StorePayload {
  return {
    vaa_bytes: "",
    status: Status.Pending,
    timestamp: new Date().toISOString(),
    retries: 0,
  };
}

export function initPayloadWithVAA(vaa_bytes: string): StorePayload {
  const sp: StorePayload = initPayload();
  sp.vaa_bytes = vaa_bytes;
  return sp;
}

export function storeKeyFromParsedVAA(
  parsedVAA: ParsedVaa<ParsedTransferPayload>
): StoreKey {
  return {
    chain_id: parsedVAA.emitterChain as number,
    emitter_address: uint8ArrayToHex(parsedVAA.emitterAddress),
    sequence: parsedVAA.sequence,
  };
}

/** Stringify the key going into redis as json */
export function storeKeyToJson(storeKey: StoreKey): string {
  return JSON.stringify(storeKey);
}

export function storeKeyFromJson(json: string): StoreKey {
  return JSON.parse(json);
}

/** Stringify the value going into redis as json */
export function storePayloadToJson(storePayload: StorePayload): string {
  return JSON.stringify(storePayload);
}

export function storePayloadFromJson(json: string): StorePayload {
  return JSON.parse(json);
}

export function resetPayload(storePayload: StorePayload): StorePayload {
  return initPayloadWithVAA(storePayload.vaa_bytes);
}

export async function clearRedis() {
  const redisClient = await connectToRedis();
  if (!redisClient) {
    logger.error("Failed to connect to redis to clear tables.");
    return;
  }
  await redisClient.FLUSHALL();
  redisClient.quit();
}

export async function demoteWorkingRedis() {
  const redisClient = await connectToRedis();
  if (!redisClient) {
    logger.error("Failed to connect to redis to clear tables.");
    return;
  }
  await redisClient.select(RedisTables.WORKING);
  for await (const si_key of redisClient.scanIterator()) {
    const si_value = await redisClient.get(si_key);
    if (!si_value) {
      continue;
    }
    logger.info("Demoting %s", si_key);
    await redisClient.del(si_key);
    await redisClient.select(RedisTables.INCOMING);
    await redisClient.set(
      si_key,
      storePayloadToJson(resetPayload(storePayloadFromJson(si_value)))
    );
    await redisClient.select(RedisTables.WORKING);
  }
  redisClient.quit();
}

type SourceToTargetMap = {
  [key in ChainId]: {
    [key in ChainId]: number;
  };
};

export function createSourceToTargetMap(
  knownChainIds: ChainId[]
): SourceToTargetMap {
  const sourceToTargetMap: SourceToTargetMap = {} as SourceToTargetMap;
  for (const sourceKey of knownChainIds) {
    sourceToTargetMap[sourceKey] = {} as { [key in ChainId]: number };
    for (const targetKey of knownChainIds) {
      sourceToTargetMap[sourceKey][targetKey] = 0;
    }
  }
  return sourceToTargetMap;
}

export async function incrementSourceToTargetMap(
  key: string,
  redisClient: RedisClientType<any>,
  parse_vaa: Function,
  sourceToTargetMap: SourceToTargetMap
): Promise<void> {
  const parsedKey = storeKeyFromJson(key);
  const si_value = await redisClient.get(key);
  if (!si_value) {
    return;
  }
  const parsedPayload = parseTransferPayload(
    Buffer.from(
      parse_vaa(hexToUint8Array(storePayloadFromJson(si_value).vaa_bytes))
        .payload
    )
  );
  if (
    sourceToTargetMap[parsedKey.chain_id as ChainId]?.[
      parsedPayload.targetChain
    ] !== undefined
  ) {
    sourceToTargetMap[parsedKey.chain_id as ChainId][
      parsedPayload.targetChain
    ]++;
  }
}

export async function monitorRedis(metrics: PromHelper) {
  const scopedLogger = getScopedLogger(["monitorRedis"], logger);
  const TEN_SECONDS: number = 10000;
  const { parse_vaa } = await importCoreWasm();
  const knownChainIds = Object.keys(chainIDStrings).map(
    (c) => Number(c) as ChainId
  );
  while (true) {
    const redisClient = await connectToRedis();
    if (!redisClient) {
      scopedLogger.error("Failed to connect to redis!");
    } else {
      try {
        await redisClient.select(RedisTables.INCOMING);
        const incomingSourceToTargetMap =
          createSourceToTargetMap(knownChainIds);
        for await (const si_key of redisClient.scanIterator()) {
          incrementSourceToTargetMap(
            si_key,
            redisClient,
            parse_vaa,
            incomingSourceToTargetMap
          );
        }
        for (const sourceKey of knownChainIds) {
          for (const targetKey of knownChainIds) {
            metrics.setRedisQueue(
              RedisTables.INCOMING,
              sourceKey,
              targetKey,
              incomingSourceToTargetMap[sourceKey][targetKey]
            );
          }
        }
        await redisClient.select(RedisTables.WORKING);
        const workingSourceToTargetMap = createSourceToTargetMap(knownChainIds);
        for await (const si_key of redisClient.scanIterator()) {
          incrementSourceToTargetMap(
            si_key,
            redisClient,
            parse_vaa,
            workingSourceToTargetMap
          );
        }
        for (const sourceKey of knownChainIds) {
          for (const targetKey of knownChainIds) {
            metrics.setRedisQueue(
              RedisTables.WORKING,
              sourceKey,
              targetKey,
              workingSourceToTargetMap[sourceKey][targetKey]
            );
          }
        }
      } catch (e) {
        scopedLogger.error("Failed to get dbSize and set metrics!");
      }
      try {
        redisClient.quit();
      } catch (e) {}
    }
    await sleep(TEN_SECONDS);
  }
}

/** Check to see if a queue is in the listener memory queue before redis */
export async function checkQueue(key: string): Promise<string | null> {
  try {
    const backupQueue = getBackupQueue();
    const queuedRecord = backupQueue.find((record) => {
      record[0] === key;
    });

    if (queuedRecord) {
      logger.debug("VAA was already in the listener queue");
      return "VAA was already in the listener queue";
    }

    const rClient = await connectToRedis();
    if (!rClient) {
      logger.error("Failed to connect to redis");
      return null;
    }
    /**
     * TODO: Pretty sure this code never ever worked for checking if a key is in redis.
     *
     * The `key` variable is `chainId + ":" + address`, but the actual redis keys come
     * from serializing the StoreKey type into a stringified json representation.
     */
    await rClient.select(RedisTables.INCOMING);
    const record1 = await rClient.get(key);

    if (record1) {
      logger.debug("VAA was already in INCOMING table");
      rClient.quit();
      return "VAA was already in INCOMING table";
    }

    await rClient.select(RedisTables.WORKING);
    const record2 = await rClient.get(key);
    if (record2) {
      logger.debug("VAA was already in WORKING table");
      rClient.quit();
      return "VAA was already in WORKING table";
    }
    rClient.quit();
  } catch (e) {
    logger.error("Failed to connect to redis");
  }

  return null;
}
