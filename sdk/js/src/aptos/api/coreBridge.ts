import { TxnBuilderTypes } from "aptos";
import { ChainId } from "../../utils";
import { AptosClientWrapper } from "../client";
import { WormholeAptosBaseApi } from "./base";

export class WormholeAptosCoreBridgeApi extends WormholeAptosBaseApi {
  constructor(client: AptosClientWrapper, address?: string) {
    super(client);
    this.address = address;
  }

  // Guardian set upgrade

  upgradeGuardianSet = (
    senderAddress: string,
    vaa: Uint8Array,
  ): Promise<TxnBuilderTypes.RawTransaction> => {
    if (!this.address) throw "Need core bridge address.";
    const payload = {
      function: `${this.address}::guardian_set_upgrade::submit_vaa`,
      type_arguments: [],
      arguments: [vaa],
    };
    return this.client.executeEntryFunction(senderAddress, payload);
  };

  // Init WH

  initWormhole = (
    senderAddress: string,
    chainId: ChainId,
    governanceChainId: number,
    governanceContract: Uint8Array,
    initialGuardian: Uint8Array,
  ): Promise<TxnBuilderTypes.RawTransaction> => {
    if (!this.address) throw "Need core bridge address.";
    const payload = {
      function: `${this.address}::wormhole::init`,
      type_arguments: [],
      arguments: [chainId, governanceChainId, governanceContract, initialGuardian],
    };
    return this.client.executeEntryFunction(senderAddress, payload);
  };
}
