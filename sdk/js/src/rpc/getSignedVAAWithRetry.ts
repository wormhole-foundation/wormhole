import { ChainId, getSignedVAA } from "..";

export async function getSignedVAAWithRetry(
  hosts: string[],
  emitterChain: ChainId,
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
        emitterChain,
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
