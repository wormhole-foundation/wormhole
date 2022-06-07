import { ChainId } from "@certusone/wormhole-sdk";

import { Relayer } from "../definitions";
import { getScopedLogger, ScopedLogger } from "../../helpers/logHelper";

import { StorePayload } from "../../helpers/redisHelper"

/** Relayer for payload 1 token bridge messages only */
export class TokenBridgeRelayer implements Relayer {
    logger: ScopedLogger
    
    payload: StorePayload | null = null
    
    constructor() {
        this.logger = getScopedLogger(["TokenBridgeRelayer"]);
        //this.payload = payload;
    }
  
    run(): void {
      this.logger.info("Starting the relayer")
    }
  
    isComplete(): boolean {
      this.logger.info("Check if relay is complete for a given vaa")
      return true
    }
  
    targetChain(): ChainId {
      return 1
    }
  }