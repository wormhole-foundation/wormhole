import { ChainId, ChainName, getSignedVAA } from "..";
import { coalesceChainId } from "../utils";

export async function getSignedVAAWithRetry(
  hosts: string[],
  emitterChain: ChainId | ChainName,
  emitterAddress: string,
  sequence: string,
  extraGrpcOpts = {},
  retryTimeout = 1000,
  retryAttempts?: number
) {
  let currentWormholeRpcHost = -1;
  const getNextRpcHost = () => ++currentWormholeRpcHost % hosts.length;
  let result;
  let attempts = 0;
  while (!result) {
    attempts++;
    await new Promise((resolve) => setTimeout(resolve, retryTimeout));
    try {
      result = await getSignedVAA(
        hosts[getNextRpcHost()],
        coalesceChainId(emitterChain),
        emitterAddress,
        sequence,
        extraGrpcOpts
      );
    } catch (e) {
      if (retryAttempts !== undefined && attempts > retryAttempts) {
        throw e;
      }
    }
  }
  return result;
}

export default getSignedVAAWithRetry;
