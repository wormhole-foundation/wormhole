import { Network } from "../../utils";
import { AptosClientWrapper } from "../client";
import { WormholeAptosCoreBridgeApi } from "./coreBridge";
import { AptosTokenBridgeApi as WormholeAptosTokenBridgeApi } from "./tokenBridge";

export class WormholeAptosApi {
  wormhole: WormholeAptosCoreBridgeApi;
  tokenBridge: WormholeAptosTokenBridgeApi;

  private client: AptosClientWrapper;

  constructor(client: AptosClientWrapper, network: Network) {
    this.client = client;
    this.wormhole = new WormholeAptosCoreBridgeApi(this.client, network);
    this.tokenBridge = new WormholeAptosTokenBridgeApi(this.client, network);
  }
}
