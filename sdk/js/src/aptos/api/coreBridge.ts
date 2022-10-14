import { AptosAccount } from "aptos";
import { ChainId, CONTRACTS, Network } from "../../utils";
import { AptosClientWrapper } from "../client";
import { WormholeAptosBaseApi } from "./base";

export class WormholeAptosCoreBridgeApi extends WormholeAptosBaseApi {
  constructor(client: AptosClientWrapper, network: Network) {
    super(client);
    this.address = CONTRACTS[network].aptos.core!;
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
    chainId: ChainId,
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
