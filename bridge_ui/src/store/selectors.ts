import { CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { ethers } from "ethers";
import { parseUnits } from "ethers/lib/utils";
import { RootState } from ".";

/*
 * Attest
 */

export const selectAttestActiveStep = (state: RootState) =>
  state.attest.activeStep;
export const selectAttestSourceChain = (state: RootState) =>
  state.attest.sourceChain;
export const selectAttestSourceAsset = (state: RootState) =>
  state.attest.sourceAsset;
export const selectAttestTargetChain = (state: RootState) =>
  state.attest.targetChain;
export const selectAttestSignedVAAHex = (state: RootState) =>
  state.attest.signedVAAHex;
export const selectAttestIsSending = (state: RootState) =>
  state.attest.isSending;
export const selectAttestIsCreating = (state: RootState) =>
  state.attest.isCreating;
export const selectAttestIsSourceComplete = (state: RootState) =>
  !!state.attest.sourceChain && !!state.attest.sourceAsset;
// TODO: check wrapped asset exists or is native attest
export const selectAttestIsTargetComplete = (state: RootState) =>
  selectAttestIsSourceComplete(state) && !!state.attest.targetChain;
export const selectAttestIsSendComplete = (state: RootState) =>
  !!selectAttestSignedVAAHex(state);
export const selectAttestShouldLockFields = (state: RootState) =>
  selectAttestIsSending(state) || selectAttestIsSendComplete(state);

/*
 * Transfer
 */

export const selectTransferActiveStep = (state: RootState) =>
  state.transfer.activeStep;
export const selectTransferSourceChain = (state: RootState) =>
  state.transfer.sourceChain;
export const selectTransferSourceAsset = (state: RootState) => {
  return state.transfer.sourceParsedTokenAccount?.mintKey || undefined;
};
export const selectTransferIsSourceAssetWormholeWrapped = (state: RootState) =>
  state.transfer.isSourceAssetWormholeWrapped;
export const selectTransferOriginChain = (state: RootState) =>
  state.transfer.originChain;
export const selectTransferOriginAsset = (state: RootState) =>
  state.transfer.originAsset;
export const selectTransferSourceParsedTokenAccount = (state: RootState) =>
  state.transfer.sourceParsedTokenAccount;
export const selectTransferSourceParsedTokenAccounts = (state: RootState) =>
  state.transfer.sourceParsedTokenAccounts;
export const selectTransferSourceBalanceString = (state: RootState) =>
  state.transfer.sourceParsedTokenAccount?.uiAmountString || "";
export const selectTransferAmount = (state: RootState) => state.transfer.amount;
export const selectTransferTargetChain = (state: RootState) =>
  state.transfer.targetChain;
export const selectTransferTargetAddressHex = (state: RootState) =>
  state.transfer.targetAddressHex;
export const selectTransferTargetAsset = (state: RootState) =>
  state.transfer.targetAsset;
export const selectTransferTargetParsedTokenAccount = (state: RootState) =>
  state.transfer.targetParsedTokenAccount;
export const selectTransferTargetBalanceString = (state: RootState) =>
  state.transfer.targetParsedTokenAccount?.uiAmountString || "";
export const selectTransferSignedVAAHex = (state: RootState) =>
  state.transfer.signedVAAHex;
export const selectTransferIsSending = (state: RootState) =>
  state.transfer.isSending;
export const selectTransferIsRedeeming = (state: RootState) =>
  state.transfer.isRedeeming;
export const selectTransferSourceError = (state: RootState) => {
  if (!state.transfer.sourceChain) {
    return "Select a source chain";
  }
  if (!state.transfer.sourceParsedTokenAccount) {
    return "Select a token";
  }
  if (!state.transfer.amount) {
    return "Enter an amount";
  }
  if (
    state.transfer.sourceChain === CHAIN_ID_SOLANA &&
    !state.transfer.sourceParsedTokenAccount.publicKey
  ) {
    return "Token account unavailable";
  }
  if (!state.transfer.sourceParsedTokenAccount.uiAmountString) {
    return "Token amount unavailable";
  }
  if (state.transfer.sourceParsedTokenAccount.decimals === 0) {
    // TODO: more advanced NFT check - also check supply and uri
    return "NFTs are not currently supported";
  }
  try {
    // these may trigger error: fractional component exceeds decimals
    if (
      parseUnits(
        state.transfer.amount,
        state.transfer.sourceParsedTokenAccount.decimals
      ).lte(0)
    ) {
      return "Amount must be greater than zero";
    }
    if (
      parseUnits(
        state.transfer.amount,
        state.transfer.sourceParsedTokenAccount.decimals
      ).gt(
        parseUnits(
          state.transfer.sourceParsedTokenAccount.uiAmountString,
          state.transfer.sourceParsedTokenAccount.decimals
        )
      )
    ) {
      return "Amount may not be greater than balance";
    }
  } catch (e) {
    if (e?.message) {
      return e.message.substring(0, e.message.indexOf("("));
    }
    return "Invalid amount";
  }
  return undefined;
};
export const selectTransferIsSourceComplete = (state: RootState) =>
  !selectTransferSourceError(state);
export const selectTransferTargetError = (state: RootState) => {
  const sourceError = selectTransferSourceError(state);
  if (sourceError) {
    return `Error in source: ${sourceError}`;
  }
  if (!state.transfer.targetChain) {
    return "Select a target chain";
  }
  if (!state.transfer.targetAsset) {
    return "Target asset unavailable. Is the token attested?";
  }
  if (
    state.transfer.targetChain === CHAIN_ID_ETH &&
    state.transfer.targetAsset === ethers.constants.AddressZero
  ) {
    return "Target asset unavailable. Is the token attested?";
  }
  if (!state.transfer.targetAddressHex) {
    return "Target account unavailable";
  }
};
export const selectTransferIsTargetComplete = (state: RootState) =>
  !selectTransferTargetError(state);
export const selectTransferIsSendComplete = (state: RootState) =>
  !!selectTransferSignedVAAHex(state);
export const selectTransferShouldLockFields = (state: RootState) =>
  selectTransferIsSending(state) || selectTransferIsSendComplete(state);

export const selectSolanaTokenMap = (state: RootState) => {
  return state.tokens.solanaTokenMap;
};
