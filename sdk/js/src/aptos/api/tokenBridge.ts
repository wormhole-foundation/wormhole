import { TxnBuilderTypes } from "aptos";
import { ChainId, ChainName, coalesceChainId, getAssetFullyQualifiedType } from "../../utils";
import { AptosClientWrapper } from "../client";
import { WormholeAptosBaseApi } from "./base";

export class AptosTokenBridgeApi extends WormholeAptosBaseApi {
  constructor(client: AptosClientWrapper, address?: string) {
    super(client);
    this.address = address;
  }

  // Attest token

  attestToken = (
    senderAddress: string,
    tokenChain: ChainId | ChainName,
    tokenAddress: string,
  ): Promise<TxnBuilderTypes.RawTransaction> => {
    if (!this.address) throw "Need token bridge address.";
    const assetType = getAssetFullyQualifiedType(
      this.address,
      coalesceChainId(tokenChain),
      tokenAddress,
    );
    if (!assetType) throw "Invalid asset address.";
    
    const payload = {
      function: `${this.address}::attest_token::attest_token_with_signer`,
      type_arguments: [assetType],
      arguments: [],
    };
    return this.client.executeEntryFunction(senderAddress, payload);
  };

  // Complete transfer

  completeTransfer = (
    senderAddress: string,
    tokenChain: ChainId | ChainName,
    tokenAddress: string,
    vaa: Uint8Array,
    feeRecipient: string,
  ): Promise<TxnBuilderTypes.RawTransaction> => {
    if (!this.address) throw "Need token bridge address.";
    const assetType = getAssetFullyQualifiedType(
      this.address,
      coalesceChainId(tokenChain),
      tokenAddress,
    );
    if (!assetType) throw "Invalid asset address.";

    const payload = {
      function: `${this.address}::complete_transfer::submit_vaa`,
      type_arguments: [assetType],
      arguments: [vaa, feeRecipient],
    };
    return this.client.executeEntryFunction(senderAddress, payload);
  };

  completeTransferWithPayload = (
    _senderAddress: string,
    _tokenChain: ChainId | ChainName,
    _tokenAddress: string,
    _vaa: Uint8Array,
  ): Promise<TxnBuilderTypes.RawTransaction> => {
    throw new Error("Completing transfers with payload is not yet supported in the sdk");
  };

  // Deploy coin

  // don't need `signer` and `&signer` in argument list because the Move VM will inject them
  deployCoin = (senderAddress: string): Promise<TxnBuilderTypes.RawTransaction> => {
    if (!this.address) throw "Need token bridge address.";
    const payload = {
      function: `${this.address}::deploy_coin::deploy_coin`,
      type_arguments: [],
      arguments: [],
    };
    return this.client.executeEntryFunction(senderAddress, payload);
  };

  // Register chain

  registerChain = (
    senderAddress: string,
    vaa: Uint8Array,
  ): Promise<TxnBuilderTypes.RawTransaction> => {
    if (!this.address) throw "Need token bridge address.";
    const payload = {
      function: `${this.address}::register_chain::submit_vaa`,
      type_arguments: [],
      arguments: [vaa],
    };
    return this.client.executeEntryFunction(senderAddress, payload);
  };

  // Transfer tokens

  transferTokens = (
    senderAddress: string,
    tokenChain: ChainId | ChainName,
    tokenAddress: string,
    amount: number | bigint,
    recipientChain: ChainId | ChainName,
    recipient: Uint8Array,
    relayerFee: number | bigint,
    wormholeFee: number | bigint,
    nonce: number | bigint,
    payload: string = "",
  ): Promise<TxnBuilderTypes.RawTransaction> => {
    if (!this.address) throw "Need token bridge address.";
    const assetType = getAssetFullyQualifiedType(
      this.address,
      coalesceChainId(tokenChain),
      tokenAddress,
    );
    if (!assetType) throw "Invalid asset address.";

    let entryFuncPayload;
    const recipientChainId = coalesceChainId(recipientChain);
    if (payload) {
      throw new Error("Transfer with payload are not yet supported in the sdk");
    } else {
      entryFuncPayload = {
        function: `${this.address}::transfer_tokens::transfer_tokens_with_signer`,
        type_arguments: [assetType],
        arguments: [amount, recipientChainId, recipient, relayerFee, wormholeFee, nonce],
      };
    }

    return this.client.executeEntryFunction(senderAddress, entryFuncPayload);
  };

  // Created wrapped coin

  createWrappedCoin = async (
    senderAddress: string,
    tokenChain: ChainId | ChainName,
    tokenAddress: string,
    vaa: Uint8Array,
  ): Promise<TxnBuilderTypes.RawTransaction> => {
    if (!this.address) throw "Need token bridge address.";

    // create coin type
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
    if (!assetType) throw "Invalid asset address.";
    await this.client.executeEntryFunction(senderAddress, createWrappedCoinTypePayload);

    // create coin
    const createWrappedCoinPayload = {
      function: `${this.address}::wrapped::create_wrapped_coin`,
      type_arguments: [assetType],
      arguments: [vaa],
    };
    return this.client.executeEntryFunction(senderAddress, createWrappedCoinPayload);
  };
}
