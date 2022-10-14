import { AptosClient } from "aptos";
import { CONTRACTS, Network } from "../../utils";
import { AptosClientWrapper } from "../client";
import { WormholeAptosCoreBridgeApi } from "./coreBridge";
import { AptosTokenBridgeApi as WormholeAptosTokenBridgeApi } from "./tokenBridge";

export class WormholeAptosApi {
  coreBridge: WormholeAptosCoreBridgeApi;
  tokenBridge: WormholeAptosTokenBridgeApi;

  private client: AptosClientWrapper;

  constructor(client: AptosClient, coreBridgeAddress?: string, tokenBridgeAddress?: string) {
    this.client = new AptosClientWrapper(client);
    this.coreBridge = new WormholeAptosCoreBridgeApi(this.client, coreBridgeAddress);
    this.tokenBridge = new WormholeAptosTokenBridgeApi(this.client, tokenBridgeAddress);
  }

  static fromNetwork = (client: AptosClient, network: Network) => {
    return new WormholeAptosApi(
      client,
      CONTRACTS[network].aptos.core,
      CONTRACTS[network].aptos.token_bridge!,
    );
  };
}
