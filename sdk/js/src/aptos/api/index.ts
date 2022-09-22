import { Network } from "../../utils";
import { AptosClientWrapper } from "../client";
import { AptosTokenBridgeApi } from "./tokenBridge";
import { AptosWormholeApi } from "./wormhole";

export class AptosApi {
  wormhole: AptosWormholeApi;
  tokenBridge: AptosTokenBridgeApi;

  private client: AptosClientWrapper;

  constructor(client: AptosClientWrapper, network: Network) {
    this.client = client;
    this.wormhole = new AptosWormholeApi(this.client, network);
    this.tokenBridge = new AptosTokenBridgeApi(this.client, network);
  }
}
