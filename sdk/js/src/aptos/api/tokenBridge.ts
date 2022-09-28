import {
  ChainId,
  ChainName,
  coalesceChainId,
  getAssetFullyQualifiedType
} from "../../utils";
import { AptosAccount, Types } from "aptos";
import { AptosClientWrapper } from "../client";
import { WormholeAptosBaseApi } from "./base";

export class AptosTokenBridgeApi extends WormholeAptosBaseApi {
  constructor(client: AptosClientWrapper, address?: string) {
    super(client);
    this.address = address;
  }

  // Attest token

  attestToken = (
    sender: AptosAccount,
    tokenChain: ChainId | ChainName,
    tokenAddress: string,
  ): Promise<Types.Transaction> => {
    if (!this.address) throw "Need token bridge address.";
    const assetContract = getAssetFullyQualifiedType(
      this.address,
      coalesceChainId(tokenChain),
      tokenAddress,
    );
    const payload = {
      function: `${this.address}::attest_token::attest_token_with_signer`,
      type_arguments: [`${assetContract}::coin::T`],
      arguments: [],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  // Complete transfer

  completeTransfer = (
    sender: AptosAccount,
    tokenChain: ChainId | ChainName,
    tokenAddress: string,
    vaa: Uint8Array,
    feeRecipient: string,
  ): Promise<Types.Transaction> => {
    if (!this.address) throw "Need token bridge address.";
    const assetType = getAssetFullyQualifiedType(
      this.address,
      coalesceChainId(tokenChain),
      tokenAddress,
    );
    const payload = {
      function: `${this.address}::complete_transfer::submit_vaa`,
      type_arguments: [assetType],
      arguments: [vaa, feeRecipient],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  completeTransferWithPayload = (
    sender: AptosAccount,
    tokenChain: ChainId | ChainName,
    tokenAddress: string,
    vaa: Uint8Array,
  ): Promise<Types.Transaction> => {
    if (!this.address) throw "Need token bridge address.";
    const assetType = getAssetFullyQualifiedType(
      this.address,
      coalesceChainId(tokenChain),
      tokenAddress,
    );
    const payload = {
      function: `${this.address}::complete_transfer_with_payload::submit_vaa`,
      type_arguments: [assetType],
      arguments: [vaa],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  // Deploy coin

  // don't need `signer` and `&signer` in argument list because the Move VM will inject them
  deployCoin = (sender: AptosAccount): Promise<Types.Transaction> => {
    if (!this.address) throw "Need token bridge address.";
    const payload = {
      function: `${this.address}::deploy_coin::deploy_coin`,
      type_arguments: [],
      arguments: [],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  // Register chain

  registerChain = (sender: AptosAccount, vaa: Uint8Array): Promise<Types.Transaction> => {
    if (!this.address) throw "Need token bridge address.";
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
    tokenChain: ChainId | ChainName,
    tokenAddress: string,
    amount: number | bigint,
    recipientChain: number | bigint,
    recipient: Uint8Array,
    relayerFee: number | bigint,
    wormholeFee: number | bigint,
    nonce: number | bigint,
  ): Promise<Types.Transaction> => {
    if (!this.address) throw "Need token bridge address.";
    const assetType = getAssetFullyQualifiedType(
      this.address,
      coalesceChainId(tokenChain),
      tokenAddress,
    );
    const payload = {
      function: `${this.address}::transfer_tokens::submit_vaa`,
      type_arguments: [assetType],
      arguments: [amount, recipientChain, recipient, relayerFee, wormholeFee, nonce],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  // Created wrapped coin

  createWrappedCoin = async (
    sender: AptosAccount,
    tokenChain: ChainId | ChainName,
    tokenAddress: string,
    vaa: Uint8Array,
  ): Promise<Types.Transaction> => {
    if (!this.address) throw "Need token bridge address.";
    const createWrappedCoinTypePayload = {
      function: `${this.address}::wrapped::create_wrapped_coin_type`,
      type_arguments: [],
      arguments: [vaa],
    };
    const assetType = getAssetFullyQualifiedType(
      this.address,
      coalesceChainId(tokenChain),
      tokenAddress,
    );
    const createWrappedCoinPayload = {
      function: `${this.address}::wrapped::create_wrapped_coin`,
      type_arguments: [assetType],
      arguments: [vaa],
    };

    // create coin type
    await this.client.executeEntryFunction(sender, createWrappedCoinTypePayload);

    // create coin
    return this.client.executeEntryFunction(sender, createWrappedCoinPayload);
  };
}
