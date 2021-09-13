import { ChainId, getSignedVAA } from "@certusone/wormhole-sdk";
import { WORMHOLE_RPC_HOSTS } from "./consts";

export let CURRENT_WORMHOLE_RPC_HOST = -1;

export const getNextRpcHost = () =>
  ++CURRENT_WORMHOLE_RPC_HOST % WORMHOLE_RPC_HOSTS.length;

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
        WORMHOLE_RPC_HOSTS[getNextRpcHost()],
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
