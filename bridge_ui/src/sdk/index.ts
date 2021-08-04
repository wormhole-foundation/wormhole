import {
  GrpcWebImpl,
  PublicrpcClientImpl,
} from "../proto/publicrpc/v1/publicrpc";
import { ChainId } from "../utils/consts";

export async function getSignedVAA(
  emitterChain: ChainId,
  emitterAddress: string,
  sequence: string
) {
  const rpc = new GrpcWebImpl("http://localhost:8080", {});
  const api = new PublicrpcClientImpl(rpc);
  // TODO: potential infinite loop, support cancellation?
  let result;
  while (!result) {
    console.log("wait 1 second");
    await new Promise((resolve) => setTimeout(resolve, 1000));
    console.log("check for signed vaa", emitterChain, emitterAddress, sequence);
    try {
      result = await api.GetSignedVAA({
        messageId: {
          emitterChain,
          emitterAddress,
          sequence,
        },
      });
      console.log(result);
    } catch (e) {
      console.log(e);
    }
  }
  return result;
}
