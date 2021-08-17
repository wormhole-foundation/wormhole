import { ChainId, getSignedVAA } from "@certusone/wormhole-sdk";
import { WORMHOLE_RPC_HOST } from "./consts";

export async function getSignedVAAWithRetry(
  emitterChain: ChainId,
  emitterAddress: string,
  sequence: string
) {
  let result;
  while (!result) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    try {
      result = await getSignedVAA(
        WORMHOLE_RPC_HOST,
        emitterChain,
        emitterAddress,
        sequence
      );
    } catch (e) {
      console.log(e);
    }
  }
  return result;
}
