import { Types } from "aptos";
import { ChainId, ChainName, coalesceChainId, getAssetFullyQualifiedType } from "../../utils";

// Attest token

export const attestToken = (
  tokenBridgeAddress: string,
  tokenChain: ChainId | ChainName,
  tokenAddress: string,
): Types.EntryFunctionPayload => {
  if (!tokenBridgeAddress) throw "Need token bridge address.";
  const assetType = getAssetFullyQualifiedType(
    tokenBridgeAddress,
    coalesceChainId(tokenChain),
    tokenAddress,
  );
  if (!assetType) throw "Invalid asset address.";
  
  return {
    function: `${tokenBridgeAddress}::attest_token::attest_token_with_signer`,
    type_arguments: [assetType],
    arguments: [],
  };
};

// Complete transfer

export const completeTransfer = (
  tokenBridgeAddress: string,
  tokenChain: ChainId | ChainName,
  tokenAddress: string,
  vaa: Uint8Array,
  feeRecipient: string,
): Types.EntryFunctionPayload => {
  if (!tokenBridgeAddress) throw "Need token bridge address.";
  const assetType = getAssetFullyQualifiedType(
    tokenBridgeAddress,
    coalesceChainId(tokenChain),
    tokenAddress,
  );
  if (!assetType) throw "Invalid asset address.";

  return {
    function: `${tokenBridgeAddress}::complete_transfer::submit_vaa`,
    type_arguments: [assetType],
    arguments: [vaa, feeRecipient],
  };
};

export const completeTransferWithPayload = (
  _tokenBridgeAddress: string,
  _tokenChain: ChainId | ChainName,
  _tokenAddress: string,
  _vaa: Uint8Array,
): Types.EntryFunctionPayload => {
  throw new Error("Completing transfers with payload is not yet supported in the sdk");
};

// Deploy coin

// don't need `signer` and `&signer` in argument list because the Move VM will inject them
export const deployCoin = (tokenBridgeAddress: string): Types.EntryFunctionPayload => {
  if (!tokenBridgeAddress) throw "Need token bridge address.";
  return {
    function: `${tokenBridgeAddress}::deploy_coin::deploy_coin`,
    type_arguments: [],
    arguments: [],
  };
};

// Register chain

export const registerChain = (
  tokenBridgeAddress: string,
  vaa: Uint8Array,
): Types.EntryFunctionPayload => {
  if (!tokenBridgeAddress) throw "Need token bridge address.";
  return {
    function: `${tokenBridgeAddress}::register_chain::submit_vaa`,
    type_arguments: [],
    arguments: [vaa],
  };
};

// Transfer tokens

export const transferTokens = (
  tokenBridgeAddress: string,
  tokenChain: ChainId | ChainName,
  tokenAddress: string,
  amount: number | bigint,
  recipientChain: ChainId | ChainName,
  recipient: Uint8Array,
  relayerFee: number | bigint,
  wormholeFee: number | bigint,
  nonce: number | bigint,
  payload: string = "",
): Types.EntryFunctionPayload => {
  if (!tokenBridgeAddress) throw "Need token bridge address.";
  const assetType = getAssetFullyQualifiedType(
    tokenBridgeAddress,
    coalesceChainId(tokenChain),
    tokenAddress,
  );
  if (!assetType) throw "Invalid asset address.";

  const recipientChainId = coalesceChainId(recipientChain);
  if (payload) {
    throw new Error("Transfer with payload are not yet supported in the sdk");
  } else {
    return {
      function: `${tokenBridgeAddress}::transfer_tokens::transfer_tokens_with_signer`,
      type_arguments: [assetType],
      arguments: [amount, recipientChainId, recipient, relayerFee, wormholeFee, nonce],
    };
  }
};

// Created wrapped coin

export const createWrappedCoinType = (
  tokenBridgeAddress: string,
  vaa: Uint8Array,
): Types.EntryFunctionPayload => {
  if (!tokenBridgeAddress) throw "Need token bridge address.";
  return {
    function: `${tokenBridgeAddress}::wrapped::create_wrapped_coin_type`,
    type_arguments: [],
    arguments: [vaa],
  };
};

export const createWrappedCoin = (
  tokenBridgeAddress: string,
  tokenChain: ChainId | ChainName,
  tokenAddress: string,
  vaa: Uint8Array,
): Types.EntryFunctionPayload => {
  if (!tokenBridgeAddress) throw "Need token bridge address.";
  const assetType = getAssetFullyQualifiedType(
    tokenBridgeAddress,
    coalesceChainId(tokenChain),
    tokenAddress,
  );
  if (!assetType) throw "Invalid asset address.";

  return {
    function: `${tokenBridgeAddress}::wrapped::create_wrapped_coin`,
    type_arguments: [assetType],
    arguments: [vaa],
  };
};
