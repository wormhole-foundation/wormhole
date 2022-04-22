import { ChainId, ChainName, coalesceChainId } from "../utils/consts";
import {
  GrpcWebImpl,
  PublicRPCServiceClientImpl,
} from "../proto/publicrpc/v1/publicrpc";

export async function getSignedVAA(
  host: string,
  emitterChain: ChainId | ChainName,
  emitterAddress: string,
  sequence: string,
  extraGrpcOpts = {}
) {
  const rpc = new GrpcWebImpl(host, extraGrpcOpts);
  const api = new PublicRPCServiceClientImpl(rpc);
  return await api.GetSignedVAA({
    messageId: {
      emitterChain: coalesceChainId(emitterChain),
      emitterAddress,
      sequence,
    },
  });
}
