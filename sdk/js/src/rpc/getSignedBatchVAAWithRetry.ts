import { ChainId, ChainName, getSignedBatchVAA } from "..";
import { coalesceChainId } from "../utils";

export async function getSignedBatchVAAWithRetry(
  hosts: string[],
  emitterChain: ChainId | ChainName,
  transactionId: Uint8Array,
  nonce: number,
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
      result = await getSignedBatchVAA(
        hosts[getNextRpcHost()],
        coalesceChainId(emitterChain),
        transactionId,
        nonce,
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

export default getSignedBatchVAAWithRetry;
