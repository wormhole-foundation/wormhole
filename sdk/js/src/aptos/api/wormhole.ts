import { AptosAccount } from "aptos";
import { CONTRACTS, Network } from "../../utils";
import { AptosClientWrapper } from "../client";
import { AptosBaseApi } from "./base";

export class AptosWormholeApi extends AptosBaseApi {
  constructor(client: AptosClientWrapper, network: Network) {
    super(client);
    this.address = CONTRACTS[network].aptos.core;
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
