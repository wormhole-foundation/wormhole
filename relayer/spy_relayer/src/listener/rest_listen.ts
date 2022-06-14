import { Request, Response } from "express";
import { getBackend } from "../backends";
import { getListenerEnvironment, ListenerEnvironment } from "../configureEnv";
import { getLogger } from "../helpers/logHelper";

let logger = getLogger();
let env: ListenerEnvironment;

export function init(runRest: boolean): boolean {
  if (!runRest) return true;
  try {
    env = getListenerEnvironment();
  } catch (e) {
    logger.error(
      "Encountered and error while initializing the listener environment: " + e
    );
    return false;
  }
  if (!env.restPort) {
    return true;
  }

  return true;
}

export async function run() {
  if (!env.restPort) return;

  const express = require("express");
  const cors = require("cors");
  const app = express();
  app.use(cors());
  app.listen(env.restPort, () =>
    logger.info("listening on REST port %d!", env.restPort)
  );

  (async () => {
    app.get("/relayvaa/:vaa", async (req: Request, res: Response) => {
      try {
        const rawVaa = Uint8Array.from(Buffer.from(req.params.vaa, "base64"));
        await getBackend().listener.process(rawVaa);

        res.status(200).json({ message: "Scheduled" });
      } catch (e) {
        logger.error(
          "failed to process rest relay of vaa request, error: %o",
          e
        );
        logger.error("offending request: %o", req);
        res.status(400).json({ message: "Request failed" });
      }
    });

    app.get("/", (req: Request, res: Response) =>
      res.json(["/relayvaa/<vaaInBase64>"])
    );
  })();
}
