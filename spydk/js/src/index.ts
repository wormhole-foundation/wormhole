import { spy, Spy } from "@certusone/wormhole-sdk-proto-node";
import grpc, { ChannelCredentials } from "@grpc/grpc-js";

const { SpyRPCServiceClient, SubscribeSignedVAARequest } = spy;

export function createSpyRPCServiceClient(
  host: string,
  credentials: grpc.ChannelCredentials = ChannelCredentials.createInsecure(),
  options?: Partial<grpc.ChannelOptions>
) {
  return new SpyRPCServiceClient(host, credentials, options);
}

export async function subscribeSignedVAA(
  client: Spy.SpyRPCServiceClient,
  request: Spy.DeepPartial<Spy.SubscribeSignedVAARequest>
) {
  return client.subscribeSignedVAA(
    SubscribeSignedVAARequest.fromPartial(request)
  );
}
