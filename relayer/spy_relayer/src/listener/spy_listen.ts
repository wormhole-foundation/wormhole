import {
  createSpyRPCServiceClient,
  subscribeSignedVAA,
} from "@certusone/wormhole-spydk";
import { getListenerEnvironment, ListenerEnvironment } from "../configureEnv";
import { getLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";
import { sleep } from "../helpers/utils";

let metrics: PromHelper;
let env: ListenerEnvironment;
let logger = getLogger();
let vaaUriPrelude: string;

export function init(runListen: boolean): boolean {
  if (!runListen) return true;
  try {
    env = getListenerEnvironment();
    vaaUriPrelude =
      "http://localhost:" +
      (process.env.REST_PORT ? process.env.REST_PORT : "4200") +
      "/relayvaa/";
  } catch (e) {
    logger.error("Error initializing listener environment: " + e);
    return false;
  }

  return true;
}

export async function run(ph: PromHelper) {
  const logger = getLogger();
  metrics = ph;
  logger.info("Attempting to run Listener...");
  logger.info(
    "spy_relay starting up, will listen for signed VAAs from [" +
      env.spyServiceHost +
      "]"
  );

  let typedFilters = await env.listenerBackend.getEmitterFilters()
  const wrappedFilters = { filters: typedFilters };

  while (true) {
    let stream: any;
    try {
      const client = createSpyRPCServiceClient(
        env.spyServiceHost || ""
      );
      stream = await subscribeSignedVAA(client, wrappedFilters);

      //TODO validate that this is the correct type of the vaaBytes
      stream.on("data", ({ vaaBytes }: { vaaBytes: Buffer }) => {
        metrics.incIncoming()
        const asUint8 = new Uint8Array(vaaBytes);
        env.listenerBackend.process(asUint8);
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

      logger.info(
        "connected to spy service, listening for transfer signed VAAs"
      );

      while (connected) {
        await sleep(1000);
      }
    } catch (e) {
      logger.error("spy service threw an exception: %o", e);
    }

    stream.end;
    await sleep(5 * 1000);
    logger.info("attempting to reconnect to the spy service");
  }
}