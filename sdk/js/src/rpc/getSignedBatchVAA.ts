import { ChainId, ChainName, coalesceChainId } from "../utils/consts";
import { publicrpc } from "@certusone/wormhole-sdk-proto-web";
const { GrpcWebImpl, PublicRPCServiceClientImpl } = publicrpc;

export async function getSignedBatchVAA(
  host: string,
  emitterChain: ChainId | ChainName,
  transactionId: Uint8Array,
  nonce: number,
  extraGrpcOpts = {}
) {
  const rpc = new GrpcWebImpl(host, extraGrpcOpts);
  const api = new PublicRPCServiceClientImpl(rpc);
  return await api.GetSignedBatchVAA({
    batchId: {
      emitterChain: coalesceChainId(emitterChain),
      txId: transactionId,
      nonce,
    },
  });
}
