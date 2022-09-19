import { AptosClientWrapper } from "../client";
import { AptosTokenBridgeApi } from "./tokenBridge";
import { AptosWormholeApi } from "./wormhole";

export class AptosApi {
  wormhole: AptosWormholeApi;
  tokenBridge: AptosTokenBridgeApi;

  private client: AptosClientWrapper;

  constructor(client: AptosClientWrapper) {
    this.client = client;
    this.wormhole = new AptosWormholeApi(this.client);
    this.tokenBridge = new AptosTokenBridgeApi(this.client);
  }
}
