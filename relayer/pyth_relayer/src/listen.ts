import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  hexToUint8Array,
  uint8ArrayToHex,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
} from "@certusone/wormhole-sdk";

import {
  createSpyRPCServiceClient,
  subscribeSignedVAA,
} from "@certusone/wormhole-spydk";

import { importCoreWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";

import * as helpers from "./helpers";
import { logger } from "./helpers";
import { postEvent } from "./worker";
import { PromHelper } from "./promHelpers";

let seqMap = new Map<string, number>();
let listenOnly: boolean = false;
let metrics: PromHelper;

export function init(lo: boolean): boolean {
  listenOnly = lo;

  if (!process.env.SPY_SERVICE_HOST) {
    logger.error("Missing environment variable SPY_SERVICE_HOST");
    return false;
  }

  return true;
}

export async function run(pm: PromHelper) {
  metrics = pm;
  logger.info(
    "pyth_relay starting up, will listen for signed VAAs from [" +
      process.env.SPY_SERVICE_HOST +
      "]"
  );

  (async () => {
    let filter = {};
    if (process.env.SPY_SERVICE_FILTERS) {
      const parsedJsonFilters = eval(process.env.SPY_SERVICE_FILTERS);

      let myFilters = [];
      for (let i = 0; i < parsedJsonFilters.length; i++) {
        let myChainId = parseInt(parsedJsonFilters[i].chain_id) as ChainId;
        let myEmitterAddress = parsedJsonFilters[i].emitter_address;
        // let myEmitterAddress = await encodeEmitterAddress(
        //   myChainId,
        //   parsedJsonFilters[i].emitter_address
        // );
        let myEmitterFilter = {
          emitterFilter: {
            chainId: myChainId,
            emitterAddress: myEmitterAddress,
          },
        };
        logger.info(
          "adding filter: chainId: [" +
            myEmitterFilter.emitterFilter.chainId +
            "], emitterAddress: [" +
            myEmitterFilter.emitterFilter.emitterAddress +
            "]"
        );
        myFilters.push(myEmitterFilter);
      }

      logger.info("setting " + myFilters.length + " filters");
      filter = {
        filters: myFilters,
      };
    } else {
      logger.info("processing all signed VAAs");
    }

    while (true) {
      let stream: any;
      try {
        const client = createSpyRPCServiceClient(
          process.env.SPY_SERVICE_HOST || ""
        );
        stream = await subscribeSignedVAA(client, filter);

        stream.on("data", ({ vaaBytes }: { vaaBytes: string }) => {
          processVaa(vaaBytes);
        });

        let connected = true;
        stream.on("error", (err: any) => {
          logger.error("spy service returned an error: %o", err);
          connected = false;
        });

        stream.on("close", () => {
          logger.error("spy service closed the connection!");
          connected = false;
        });

        logger.info("connected to spy service, listening for messages");

        while (connected) {
          await helpers.sleep(1000);
        }
      } catch (e) {
        logger.error("spy service threw an exception: %o", e);
      }

      stream.end;
      await helpers.sleep(5 * 1000);
      logger.info("attempting to reconnect to the spy service");
    }
  })();
}

async function encodeEmitterAddress(
  myChainId: ChainId,
  emitterAddressStr: string
): Promise<string> {
  if (myChainId === CHAIN_ID_SOLANA) {
    return await getEmitterAddressSolana(emitterAddressStr);
  }

  if (myChainId === CHAIN_ID_TERRA) {
    return await getEmitterAddressTerra(emitterAddressStr);
  }

  return getEmitterAddressEth(emitterAddressStr);
}

async function processVaa(vaaBytes: string) {
  let receiveTime = new Date();
  const { parse_vaa } = await importCoreWasm();
  const parsedVAA = parse_vaa(hexToUint8Array(vaaBytes));
  // logger.debug(
  //   "processVaa, vaa len: " +
  //     vaaBytes.length +
  //     ", payload len: " +
  //     parsedVAA.payload.length
  // );

  // logger.debug("listen:processVaa: parsedVAA: %o", parsedVAA);

  if (isPyth(parsedVAA.payload)) {
    if (parsedVAA.payload.length < helpers.PYTH_PRICE_ATTESTATION_LENGTH) {
      logger.error(
        "dropping vaa because the payload length is wrong: length: " +
          parsedVAA.payload.length +
          ", expected length:",
        helpers.PYTH_PRICE_ATTESTATION_LENGTH + ", vaa: %o",
        parsedVAA
      );
      return;
    }

    let pa = helpers.parsePythPriceAttestation(Buffer.from(parsedVAA.payload));
    // logger.debug("listen:processVaa: price attestation: %o", pa);

    let key = pa.priceId;
    let lastSeqNum = seqMap.get(key);
    if (lastSeqNum) {
      if (lastSeqNum >= parsedVAA.sequence) {
        logger.debug(
          "ignoring duplicate: emitter: [" +
            parsedVAA.emitter_chain +
            ":" +
            uint8ArrayToHex(parsedVAA.emitter_address) +
            "], productId: [" +
            pa.productId +
            "], priceId: [" +
            pa.priceId +
            "], seqNum: " +
            parsedVAA.sequence
        );
        return;
      }
    }

    seqMap.set(key, parsedVAA.sequence);

    logger.info(
      "received: emitter: [" +
        parsedVAA.emitter_chain +
        ":" +
        uint8ArrayToHex(parsedVAA.emitter_address) +
        "], seqNum: " +
        parsedVAA.sequence +
        ", productId: [" +
        pa.productId +
        "], priceId: [" +
        pa.priceId +
        "], priceType: " +
        pa.priceType +
        ", price: " +
        pa.price +
        ", exponent: " +
        pa.exponent +
        ", confidenceInterval: " +
        pa.confidenceInterval +
        ", timeStamp: " +
        pa.timestamp +
        ", computedPrice: " +
        helpers.computePrice(pa.price, pa.exponent) +
        " +/-" +
        helpers.computePrice(pa.confidenceInterval, pa.exponent)
      // +
      // ", payload: [" +
      // uint8ArrayToHex(parsedVAA.payload) +
      // "]"
    );

    metrics.incIncoming();
    if (!listenOnly) {
      logger.debug("posting to worker");
      await postEvent(vaaBytes, pa, parsedVAA.sequence, receiveTime);
    }
  } else {
    logger.debug(
      "dropping non-pyth vaa, payload type " +
        parsedVAA.payload[0] +
        ", vaa: %o",
      parsedVAA
    );
  }
}

function isPyth(payload: Buffer): boolean {
  if (payload.length < 4) return false;
  if (
    payload[0] === 80 &&
    payload[1] === 50 &&
    payload[2] === 87 &&
    payload[3] === 72
  ) {
    // P2WH
    return true;
  }

  return false;
}
