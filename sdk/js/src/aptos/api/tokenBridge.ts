import { AptosAccount } from "aptos";
import { AptosClientWrapper } from "../client";

export class AptosTokenBridgeApi {
  client: AptosClientWrapper;

  constructor(client: AptosClientWrapper) {
    this.client = client;
  }

  attestToken = (sender: AptosAccount, feeCoins: any) => {
    // return this.client.executeTransaction(sender, { feeCoins });
  };
}
