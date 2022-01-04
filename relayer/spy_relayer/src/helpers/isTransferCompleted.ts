import {
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  getIsTransferCompletedEth,
  isEVMChain,
} from "@certusone/wormhole-sdk";
import { getCommonEnvironment } from "../configureEnv";
import { parseVaaTyped, parseTransferPayload } from "../listener/validation";
import { getLogger } from "./logHelper";

const logger = getLogger();
const commonEnv = getCommonEnvironment();

//TODO get all the needed things here onto the common env for this function.
export async function getIsTransferCompleted(signedVAA: Uint8Array) {
  try {
    const vaa = await parseVaaTyped(signedVAA);
    const payload = parseTransferPayload(vaa.payload);
    const chain = payload.targetChain;

    if (isEVMChain(chain)) {
      //getIsTransferCompletedEth()
    } else if (chain === CHAIN_ID_SOLANA) {
      //
    } else if (chain === CHAIN_ID_TERRA) {
      //
    } else {
      return false;
    }
  } catch (e) {
    return false;
  }
}
