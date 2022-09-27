import { AptosAccount } from "aptos";
import { AptosClientWrapper } from "../client";

export class WormholeAptosBaseApi {
  protected client: AptosClientWrapper;
  protected address?: string;

  constructor(client: AptosClientWrapper) {
    this.client = client;
  }

  // Contract upgrade

  authorizeUpgrade = (sender: AptosAccount, vaa: Uint8Array): Promise<string> => {
    if (!this.address) throw "Need bridge address.";
    const payload = {
      function: `${this.address}::contract_upgrade::submit_vaa`,
      type_arguments: [],
      arguments: [vaa],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  upgradeContract = (
    sender: AptosAccount,
    metadataSerialized: Uint8Array,
    code: Array<Uint8Array>,
  ): Promise<string> => {
    if (!this.address) throw "Need bridge address.";
    const payload = {
      function: `${this.address}::contract_upgrade::upgrade`,
      type_arguments: [],
      arguments: [metadataSerialized, code],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  migrateContract = (sender: AptosAccount): Promise<string> => {
    if (!this.address) throw "Need bridge address.";
    const payload = {
      function: `${this.address}::contract_upgrade::migrate`,
      type_arguments: [],
      arguments: [],
    };
    return this.client.executeEntryFunction(sender, payload);
  };
}
