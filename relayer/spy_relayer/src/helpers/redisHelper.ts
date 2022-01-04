import { createClient, RedisClientType } from "redis";
import { getCommonEnvironment } from "../configureEnv";
import { Mutex } from "async-mutex";
import { getLogger } from "./logHelper";
import { ChainId } from "@certusone/wormhole-sdk";
import { connect } from "http2";

import { uint8ArrayToHex } from "@certusone/wormhole-sdk";
import { ParsedTransferPayload, ParsedVaa } from "../listener/validation";

const logger = getLogger();
const commonEnv = getCommonEnvironment();
const { redisHost, redisPort } = commonEnv;

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

export function init(): boolean {
  logger.info("will connect to redis at [" + redisHost + ":" + redisPort + "]");

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
    logger.error("storeInRedis: invalid name");
    return;
  }
  if (!value) {
    logger.error("storeInRedis: invalid value");
    return;
  }

  await redisMutex.runExclusive(async () => {
    logger.debug("storeInRedis: connecting to redis.");
    const redisClient = await connectToRedis();
    if (!redisClient) {
      redisQueue.push([name, value]);
      logger.error(
        "Failed to connect to redis, enqueued vaa, there are now " +
          redisQueue.length +
          " enqueued events"
      );
      return;
    }

    if (redisQueue.length !== 0) {
      logger.info(
        "now connected to redis, playing out " +
          redisQueue.length +
          " enqueued events"
      );
      for (let idx = 0; idx < redisQueue.length; ++idx) {
        await addToRedis(redisClient, redisQueue[idx][0], redisQueue[idx][1]);
      }
      redisQueue = [];
    }

    await addToRedis(redisClient, name, value);
    await redisClient.quit();
  });
}

export async function addToRedis(
  redisClient: any,
  name: string,
  value: string
) {
  try {
    logger.debug("storeInRedis: storing in redis.");
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

export function initPayload(): StorePayload {
  return {
    vaa_bytes: "",
    status: Status.Pending,
    timestamp: Date().toString(),
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

export function storeKeyToJson(storeKey: StoreKey): string {
  return JSON.stringify(storeKey);
}

export function storeKeyFromJson(json: string): StoreKey {
  return JSON.parse(json);
}

export function storePayloadToJson(storePayload: StorePayload): string {
  return JSON.stringify(storePayload);
}

export function storePayloadFromJson(json: string): StorePayload {
  return JSON.parse(json);
}

export async function pushVaaToRedis(
  parsedVAA: ParsedVaa<ParsedTransferPayload>,
  hexVaa: string
) {
  const transferPayload = parsedVAA.payload;

  logger.info(
    "forwarding vaa to relayer: emitter: [" +
      parsedVAA.emitterChain +
      ":" +
      uint8ArrayToHex(parsedVAA.emitterAddress) +
      "], seqNum: " +
      parsedVAA.sequence +
      ", payload: origin: [" +
      transferPayload.originAddress +
      ":" +
      transferPayload.originAddress +
      "], target: [" +
      transferPayload.targetChain +
      ":" +
      transferPayload.targetAddress +
      "],  amount: " +
      transferPayload.amount +
      "],  fee: " +
      transferPayload.fee +
      ", "
  );
  const storeKey = storeKeyFromParsedVAA(parsedVAA);
  const storePayload = initPayloadWithVAA(hexVaa);

  logger.debug(
    "storing: key: [" +
      storeKey.chain_id +
      "/" +
      storeKey.emitter_address +
      "/" +
      storeKey.sequence +
      "], payload: [" +
      storePayloadToJson(storePayload) +
      "]"
  );

  await storeInRedis(
    storeKeyToJson(storeKey),
    storePayloadToJson(storePayload)
  );
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
