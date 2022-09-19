import { AptosClient } from "aptos";

export class AptosClientWrapper {
  client: AptosClient;

  constructor(nodeUrl: string) {
    this.client = new AptosClient(nodeUrl);
  }
}
