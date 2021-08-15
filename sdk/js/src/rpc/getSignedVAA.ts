import { ChainId } from "../utils/consts";
import {
  GrpcWebImpl,
  PublicrpcClientImpl,
} from "../proto/publicrpc/v1/publicrpc";

export async function getSignedVAA(
  emitterChain: ChainId,
  emitterAddress: string,
  sequence: string
) {
  const rpc = new GrpcWebImpl("http://localhost:8080", {}); // TODO: make this a parameter
  const api = new PublicrpcClientImpl(rpc);
  // TODO: move this loop outside sdk
  let result;
  while (!result) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    try {
      result = await api.GetSignedVAA({
        messageId: {
          emitterChain,
          emitterAddress,
          sequence,
        },
      });
    } catch (e) {
      // TODO: instead of try/catch, simply return api.GetSignedVAA
    }
  }
  return result;
}
