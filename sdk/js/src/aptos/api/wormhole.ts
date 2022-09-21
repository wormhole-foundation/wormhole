import { AptosAccount } from "aptos";
import { AptosClientWrapper } from "../client";
import { AptosNetwork, CONTRACT_ADDRESSES } from "../constants";
import { AptosBaseApi } from "./base";

export class AptosWormholeApi extends AptosBaseApi {
  constructor(client: AptosClientWrapper, network: AptosNetwork) {
    super(client);
    this.address = CONTRACT_ADDRESSES[network].core;
  }

  // Guardian set upgrade

  upgradeGuardianSet = (sender: AptosAccount, vaa: Uint8Array): Promise<string> => {
    const payload = {
      function: `${this.address}::guardian_set_upgrade::submit_vaa`,
      type_arguments: [],
      arguments: [vaa],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  // Init WH

  initWormhole = (
    sender: AptosAccount,
    chainId: number,
    governanceChainId: number,
    governanceContract: Uint8Array,
    initialGuardian: Uint8Array,
  ): Promise<string> => {
    const payload = {
      function: `${this.address}::wormhole::init`,
      type_arguments: [],
      arguments: [chainId, governanceChainId, governanceContract, initialGuardian],
    };
    return this.client.executeEntryFunction(sender, payload);
  };
}
