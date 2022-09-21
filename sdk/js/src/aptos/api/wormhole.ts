import { AptosAccount } from 'aptos';
import { AptosClientWrapper } from '../client';

export class AptosWormholeApi {
  private client: AptosClientWrapper;
  private address: string;

  constructor(client: AptosClientWrapper, network: string) {
    this.client = client;
    this.address = '';
  }

  // Contract upgrade

  authorizeUpgrade = (sender: AptosAccount, vaa: Uint8Array) => {
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
  ) => {
    const payload = {
      function: `${this.address}::contract_upgrade::upgrade`,
      type_arguments: [],
      arguments: [metadataSerialized, code],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  migrateContract = (sender: AptosAccount) => {
    const payload = {
      function: `${this.address}::contract_upgrade::migrate`,
      type_arguments: [],
      arguments: [],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  // Guardian set upgrade

  upgradeGuardianSet = (sender: AptosAccount, vaa: Uint8Array) => {
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
  ) => {
    const payload = {
      function: `${this.address}::wormhole::init`,
      type_arguments: [],
      arguments: [chainId, governanceChainId, governanceContract, initialGuardian],
    };
    return this.client.executeEntryFunction(sender, payload);
  };
}
