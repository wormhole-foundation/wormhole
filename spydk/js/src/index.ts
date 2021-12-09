import {
    DeepPartial,
    SpyRPCServiceClient,
    SubscribeSignedVAARequest,
  } from "./proto/spy/v1/spy";
  import grpc, { ChannelCredentials } from "@grpc/grpc-js";
  
  export function createSpyRPCServiceClient(
    host: string,
    credentials: grpc.ChannelCredentials = ChannelCredentials.createInsecure(),
    options?: Partial<grpc.ChannelOptions>
  ) {
    return new SpyRPCServiceClient(host, credentials, options);
  }
  
  export async function subscribeSignedVAA(
    client: SpyRPCServiceClient,
    request: DeepPartial<SubscribeSignedVAARequest>
  ) {
    return client.subscribeSignedVAA(
      SubscribeSignedVAARequest.fromPartial(request)
    );
  }
  