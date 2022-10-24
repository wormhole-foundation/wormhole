import { AptosAccount } from "aptos";
import {
  ChainId,
  ChainName,
  coalesceChainId,
  CONTRACTS,
  getAssetFullyQualifiedType,
  Network,
} from "../../utils";
import { AptosClientWrapper } from "../client";
import { WormholeAptosBaseApi } from "./base";

export class AptosTokenBridgeApi extends WormholeAptosBaseApi {
  constructor(client: AptosClientWrapper, network: Network) {
    super(client);
    this.address = CONTRACTS[network].aptos.token_bridge!;
  }

  // Complete transfer

  completeTransfer = (
    sender: AptosAccount,
    tokenChain: ChainId | ChainName,
    tokenAddress: string,
    vaa: Uint8Array,
    feeRecipient: string,
  ): Promise<string> => {
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
  ): Promise<string> => {
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
  deployCoin = (sender: AptosAccount): Promise<string> => {
    const payload = {
      function: `${this.address}::deploy_coin::deploy_coin`,
      type_arguments: [],
      arguments: [],
    };
    return this.client.executeEntryFunction(sender, payload);
  };

  // Register chain

  registerChain = (sender: AptosAccount, vaa: Uint8Array): Promise<string> => {
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
  ): Promise<string> => {
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
  ): Promise<string> => {
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
