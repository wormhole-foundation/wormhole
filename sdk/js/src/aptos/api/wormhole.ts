import { AptosClientWrapper } from "../client";

export class AptosWormholeApi {
  client: AptosClientWrapper;
  
  constructor(client: AptosClientWrapper) {
    this.client = client;
  }
}
