import { Types } from "aptos";
import { _parseVAAAlgorand } from "../../algorand";
import { assertChain, ChainId, ChainName, CHAIN_ID_APTOS, coalesceChainId, getAssetFullyQualifiedType } from "../../utils";

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
    function: `${tokenBridgeAddress}::attest_token::attest_token_entry`,
    type_arguments: [assetType],
    arguments: [],
  };
};

// Complete transfer

export const completeTransfer = (
  tokenBridgeAddress: string,
  transferVAA: Uint8Array,
  feeRecipient: string,
): Types.EntryFunctionPayload => {
  if (!tokenBridgeAddress) throw "Need token bridge address.";

  const parsedVAA = _parseVAAAlgorand(transferVAA);
  if (!parsedVAA.FromChain || !parsedVAA.Contract || !parsedVAA.ToChain) {
    throw "VAA does not contain required information";
  }

  if (parsedVAA.ToChain !== CHAIN_ID_APTOS) {
    throw "Transfer is not destined for Aptos";
  }
  
  assertChain(parsedVAA.FromChain);
  const assetType = getAssetFullyQualifiedType(
    tokenBridgeAddress,
    coalesceChainId(parsedVAA.FromChain),
    parsedVAA.Contract,
  );
  if (!assetType) throw "Invalid asset address.";

  return {
    function: `${tokenBridgeAddress}::complete_transfer::submit_vaa_entry`,
    type_arguments: [assetType],
    arguments: [transferVAA, feeRecipient],
  };
};

export const completeTransferAndRegister = (
  tokenBridgeAddress: string,
  transferVAA: Uint8Array,
): Types.EntryFunctionPayload => {
  if (!tokenBridgeAddress) throw "Need token bridge address.";

  const parsedVAA = _parseVAAAlgorand(transferVAA);
  if (!parsedVAA.FromChain || !parsedVAA.Contract || !parsedVAA.ToChain) {
    throw "VAA does not contain required information";
  }

  if (parsedVAA.ToChain !== CHAIN_ID_APTOS) {
    throw "Transfer is not destined for Aptos";
  }
  
  assertChain(parsedVAA.FromChain);
  const assetType = getAssetFullyQualifiedType(
    tokenBridgeAddress,
    coalesceChainId(parsedVAA.FromChain),
    parsedVAA.Contract,
  );
  if (!assetType) throw "Invalid asset address.";

  return {
    function: `${tokenBridgeAddress}::complete_transfer::submit_vaa_and_register_entry`,
    type_arguments: [assetType],
    arguments: [transferVAA],
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
    function: `${tokenBridgeAddress}::register_chain::submit_vaa_entry`,
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
      function: `${tokenBridgeAddress}::transfer_tokens::transfer_tokens_entry`,
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
  attestVAA: Uint8Array
): Types.EntryFunctionPayload => {
  if (!tokenBridgeAddress) throw "Need token bridge address.";

  const parsedVAA = _parseVAAAlgorand(attestVAA);
  if (!parsedVAA.FromChain || !parsedVAA.Contract) {
    throw "VAA does not contain required information";
  }

  assertChain(parsedVAA.FromChain);
  const assetType = getAssetFullyQualifiedType(
    tokenBridgeAddress,
    coalesceChainId(parsedVAA.FromChain),
    parsedVAA.Contract
  );
  if (!assetType) throw "Invalid asset address.";

  return {
    function: `${tokenBridgeAddress}::wrapped::create_wrapped_coin`,
    type_arguments: [assetType],
    arguments: [attestVAA],
  };
};
