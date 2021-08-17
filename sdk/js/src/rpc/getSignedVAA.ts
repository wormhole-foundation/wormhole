import { ChainId } from "../utils/consts";
import {
  GrpcWebImpl,
  PublicrpcClientImpl,
} from "../proto/publicrpc/v1/publicrpc";

export async function getSignedVAA(
  host: string,
  emitterChain: ChainId,
  emitterAddress: string,
  sequence: string
) {
  const rpc = new GrpcWebImpl(host, {});
  const api = new PublicrpcClientImpl(rpc);
  return await api.GetSignedVAA({
    messageId: {
      emitterChain,
      emitterAddress,
      sequence,
    },
  });
}
