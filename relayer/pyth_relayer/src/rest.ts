import { Request, Response } from "express";
import { logger } from "./helpers";
import { getStatus, getPriceData } from "./worker";

let restPort: number = 0;

export function init(runRest: boolean): boolean {
  if (!runRest) return true;
  if (!process.env.REST_PORT) return true;

  restPort = parseInt(process.env.REST_PORT);
  return true;
}

export async function run() {
  if (restPort == 0) return;

  const express = require("express");
  const cors = require("cors");
  const app = express();
  app.use(cors());

  app.listen(restPort, () =>
    logger.debug("listening on REST port " + restPort)
  );

  (async () => {
    app.get("/status", async (req: Request, res: Response) => {
      let result = await getStatus();
      res.json(result);
    });

    app.get(
      "/queryterra/:product_id/:price_id",
      async (req: Request, res: Response) => {
        let result = await getPriceData(
          req.params.product_id,
          req.params.price_id
        );
        res.json(result);
      }
    );

    app.get("/", (req: Request, res: Response) =>
      res.json(["/status", "/queryterra/<product_id>/<price_id>"])
    );
  })();
}
