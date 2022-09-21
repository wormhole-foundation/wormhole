import { AptosAccount } from "aptos";
import { assertChain } from "../../utils";
import { AptosClientWrapper } from "../client";
import { deriveWrappedAssetAddress } from "../utils";
import { AptosBaseApi } from "./base";

export class AptosTokenBridgeApi extends AptosBaseApi {
  constructor(client: AptosClientWrapper, network: string) {
    super(client, network);
  }

  // Complete transfer

  completeTransfer = (
    sender: AptosAccount,
    tokenChain: number,
    tokenAddress: string,
    vaa: Uint8Array,
    feeRecipient: string,
  ) => {
    assertChain(tokenChain);
    const assetContract = deriveWrappedAssetAddress(this.address, tokenChain, tokenAddress);
    const payload = {
      function: `${this.address}::complete_transfer::submit_vaa`,
      type_arguments: [`${assetContract}::coin::T`],
      arguments: [vaa, feeRecipient],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  completeTransferWithPayload = (
    sender: AptosAccount,
    tokenChain: number,
    tokenAddress: string,
    vaa: Uint8Array,
  ) => {
    assertChain(tokenChain);
    const assetContract = deriveWrappedAssetAddress(this.address, tokenChain, tokenAddress);
    const payload = {
      function: `${this.address}::complete_transfer_with_payload::submit_vaa`,
      type_arguments: [`${assetContract}::coin::T`],
      arguments: [vaa],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  // Deploy coin

  // don't need `signer` and `&signer` in argument list because the Move VM will inject them
  deployCoin = (sender: AptosAccount) => {
    const payload = {
      function: `${this.address}::deploy_coin::deploy_coin`,
      type_arguments: [],
      arguments: [],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  // Register chain

  registerChain = (sender: AptosAccount, vaa: Uint8Array) => {
    const payload = {
      function: `${this.address}::register_chain::submit_vaa`,
      type_arguments: [],
      arguments: [vaa],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  // Transfer tokens

  transferTokensWithSigner = (
    sender: AptosAccount,
    tokenChain: number,
    tokenAddress: string,
    amount: number | bigint,
    recipientChain: number | bigint,
    recipient: Uint8Array,
    relayerFee: number | bigint,
    wormholeFee: number | bigint,
    nonce: number | bigint,
  ) => {
    assertChain(tokenChain);
    const assetContract = deriveWrappedAssetAddress(this.address, tokenChain, tokenAddress);
    const payload = {
      function: `${this.address}::transfer_tokens::submit_vaa`,
      type_arguments: [`${assetContract}::coin::T`],
      arguments: [amount, recipientChain, recipient, relayerFee, wormholeFee, nonce],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  // Created wrapped coin

  createWrappedCoinType = (sender: AptosAccount, vaa: Uint8Array) => {
    const payload = {
      function: `${this.address}::wrapped::create_wrapped_coin_type`,
      type_arguments: [],
      arguments: [vaa],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  createWrappedCoin = (
    sender: AptosAccount,
    tokenChain: number,
    tokenAddress: string,
    vaa: Uint8Array,
  ) => {
    assertChain(tokenChain);
    const assetContract = deriveWrappedAssetAddress(this.address, tokenChain, tokenAddress);
    const payload = {
      function: `${this.address}::wrapped::create_wrapped_coin`,
      type_arguments: [`${assetContract}::coin::T`],
      arguments: [vaa],
    };
    return this.client.executeEntryFunction(sender, payload);
  };
}
