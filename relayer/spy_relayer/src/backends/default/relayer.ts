import {
  ChainId,
  hexToUint8Array,
  importCoreWasm,
  parseTransferPayload,
} from "@certusone/wormhole-sdk";

import { Relayer } from "../definitions";
import { getScopedLogger, ScopedLogger } from "../../helpers/logHelper";
import {
  connectToRedis,
  RedisTables,
  RelayResult,
  Status,
  StorePayload,
  storePayloadFromJson,
  storePayloadToJson,
} from "../../helpers/redisHelper";
import { relay } from "../../relayer/relay";
import { PromHelper } from "../../helpers/promHelpers";

let metrics: PromHelper;

/** Relayer for payload 1 token bridge messages only */
export class TokenBridgeRelayer implements Relayer {
  /** Process the relay request */
  async process(
    key: string,
    privateKey: any,
    relayLogger: ScopedLogger
  ): Promise<void> {
    const logger = getScopedLogger(["TokenBridgeRelayer.process"], relayLogger);
    try {
      logger.debug("Processing request %s...", key);
      // Get the entry from the working store
      const redisClient = await connectToRedis();
      if (!redisClient) {
        logger.error("Failed to connect to Redis in processRequest");
        return;
      }
      await redisClient.select(RedisTables.WORKING);
      let value: string | null = await redisClient.get(key);
      if (!value) {
        logger.error("Could not find key %s", key);
        return;
      }
      let payload: StorePayload = storePayloadFromJson(value);
      if (payload.status !== Status.Pending) {
        logger.info("This key %s has already been processed.", key);
        return;
      }
      // Actually do the processing here and update status and time field
      let relayResult: RelayResult;
      try {
        if (payload.retries > 0) {
          logger.info(
            "Calling with vaa_bytes %s, retry %d",
            payload.vaa_bytes,
            payload.retries
          );
        } else {
          logger.info("Calling with vaa_bytes %s", payload.vaa_bytes);
        }
        relayResult = await relay(
          payload.vaa_bytes,
          false,
          privateKey,
          logger,
          metrics
        );
        logger.info("Relay returned: %o", Status[relayResult.status]);
      } catch (e: any) {
        if (e.message) {
          logger.error("Failed to relay transfer vaa: %s", e.message);
        } else {
          logger.error("Failed to relay transfer vaa: %o", e);
        }

        relayResult = {
          status: Status.Error,
          result: e && e?.message !== undefined ? e.message : "Failure",
        };
      }

      const MAX_RETRIES = 10;
      // ChainId 0 denotes an undefined chain
      let targetChain: ChainId = 0;
      try {
        const { parse_vaa } = await importCoreWasm();
        const parsedVAA = parse_vaa(hexToUint8Array(payload.vaa_bytes));
        const transferPayload = parseTransferPayload(
          Buffer.from(parsedVAA.payload)
        );
        targetChain = transferPayload.targetChain;
      } catch (e) {}
      let retry: boolean = false;
      if (relayResult.status !== Status.Completed) {
        metrics.incFailures(targetChain);
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
      payload.timestamp = new Date().toISOString();
      payload.retries++;
      value = storePayloadToJson(payload);
      if (!retry || payload.retries > MAX_RETRIES) {
        await redisClient.set(key, value);
      } else {
        // Remove from the working table
        await redisClient.del(key);
        // Put this back into the incoming table
        await redisClient.select(RedisTables.INCOMING);
        await redisClient.set(key, value);
      }
      await redisClient.quit();
    } catch (e: any) {
      logger.error("Unexpected error in processRequest: " + e.message);
      logger.error("request key: " + key);
      logger.error(e);
    }
  }
  /** Check if the relay is completed and can't be rolled back due to a chain re-organization */
  isComplete(): boolean {
    return true;
  }

  /** Parse the target chain id from the payload */
  targetChainId(): ChainId {
    return 1;
  }
}
