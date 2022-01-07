import { CHAIN_ID_SOLANA, isEVMChain } from "@certusone/wormhole-sdk";
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
export const selectAttestAttestTx = (state: RootState) => state.attest.attestTx;
export const selectAttestSignedVAAHex = (state: RootState) =>
  state.attest.signedVAAHex;
export const selectAttestIsSending = (state: RootState) =>
  state.attest.isSending;
export const selectAttestIsCreating = (state: RootState) =>
  state.attest.isCreating;
export const selectAttestCreateTx = (state: RootState) => state.attest.createTx;
export const selectAttestIsSourceComplete = (state: RootState) =>
  !!state.attest.sourceChain && !!state.attest.sourceAsset;
// TODO: check wrapped asset exists or is native attest
export const selectAttestIsTargetComplete = (state: RootState) =>
  selectAttestIsSourceComplete(state) && !!state.attest.targetChain;
export const selectAttestIsSendComplete = (state: RootState) =>
  !!selectAttestSignedVAAHex(state);
export const selectAttestIsCreateComplete = (state: RootState) =>
  !!selectAttestCreateTx(state);
export const selectAttestShouldLockFields = (state: RootState) =>
  selectAttestIsSending(state) || selectAttestIsSendComplete(state);

/*
 * NFT
 */

export const selectNFTActiveStep = (state: RootState) => state.nft.activeStep;
export const selectNFTSourceChain = (state: RootState) => state.nft.sourceChain;
export const selectNFTSourceAsset = (state: RootState) => {
  return state.nft.sourceParsedTokenAccount?.mintKey || undefined;
};
export const selectNFTIsSourceAssetWormholeWrapped = (state: RootState) =>
  state.nft.isSourceAssetWormholeWrapped;
export const selectNFTOriginChain = (state: RootState) => state.nft.originChain;
export const selectNFTOriginAsset = (state: RootState) => state.nft.originAsset;
export const selectNFTOriginTokenId = (state: RootState) =>
  state.nft.originTokenId;
export const selectNFTSourceWalletAddress = (state: RootState) =>
  state.nft.sourceWalletAddress;
export const selectNFTSourceParsedTokenAccount = (state: RootState) =>
  state.nft.sourceParsedTokenAccount;
export const selectNFTSourceParsedTokenAccounts = (state: RootState) =>
  state.nft.sourceParsedTokenAccounts;
export const selectNFTSourceBalanceString = (state: RootState) =>
  state.nft.sourceParsedTokenAccount?.uiAmountString || "";
export const selectNFTTargetChain = (state: RootState) => state.nft.targetChain;
export const selectNFTTargetAddressHex = (state: RootState) =>
  state.nft.targetAddressHex;
export const selectNFTTargetAsset = (state: RootState) =>
  state.nft.targetAsset.data?.address;
export const selectNFTTransferTx = (state: RootState) => state.nft.transferTx;
export const selectNFTSignedVAAHex = (state: RootState) =>
  state.nft.signedVAAHex;
export const selectNFTIsSending = (state: RootState) => state.nft.isSending;
export const selectNFTIsRedeeming = (state: RootState) => state.nft.isRedeeming;
export const selectNFTRedeemTx = (state: RootState) => state.nft.redeemTx;
export const selectNFTSourceError = (state: RootState): string | undefined => {
  if (!state.nft.sourceChain) {
    return "Select a source chain";
  }
  if (!state.nft.sourceParsedTokenAccount) {
    return "Select an NFT";
  }
  if (
    state.nft.sourceChain === CHAIN_ID_SOLANA &&
    !state.nft.sourceParsedTokenAccount.publicKey
  ) {
    return "Token account unavailable";
  }
  if (!state.nft.sourceParsedTokenAccount.uiAmountString) {
    return "Token amount unavailable";
  }
  if (state.nft.sourceParsedTokenAccount.decimals !== 0) {
    // TODO: more advanced NFT check - also check supply and uri
    return "For non-NFTs, use the Transfer flow";
  }
  if (
    state.nft.sourceParsedTokenAccount?.uri === null ||
    state.nft.sourceParsedTokenAccount?.uri === undefined
  ) {
    return "Failed to load NFT Metadata.";
  }
  if (
    state.nft.sourceParsedTokenAccount?.uri &&
    state.nft.sourceParsedTokenAccount?.uri.length > 200
  ) {
    return "This NFT has a URL longer than the maximum supported length of 200.";
  }
  try {
    // these may trigger error: fractional component exceeds decimals
    if (
      parseUnits(
        state.nft.sourceParsedTokenAccount.uiAmountString,
        state.nft.sourceParsedTokenAccount.decimals
      ).lte(0)
    ) {
      return "Balance must be greater than zero";
    }
  } catch (e: any) {
    if (e?.message) {
      return e.message.substring(0, e.message.indexOf("("));
    }
    return "Invalid amount";
  }
  return undefined;
};
export const selectNFTIsSourceComplete = (state: RootState) =>
  !selectNFTSourceError(state);
export const selectNFTTargetError = (state: RootState) => {
  const sourceError = selectNFTSourceError(state);
  if (sourceError) {
    return `Error in source: ${sourceError}`;
  }
  if (!state.nft.targetChain) {
    return "Select a target chain";
  }
  if (state.nft.sourceChain === state.nft.targetChain) {
    return "Select a different target and source";
  }
  if (
    state.nft.targetChain === CHAIN_ID_SOLANA &&
    !selectNFTTargetAsset(state)
  ) {
    // target asset is only required for solana
    // in the cases of new transfers, target asset will not exist and be created on redeem
    // Solana requires the derived address to derive the associated token account which is the target on the vaa
    return UNREGISTERED_ERROR_MESSAGE;
  }
  if (!state.nft.targetAddressHex) {
    return "Target account unavailable";
  }
};
export const selectNFTIsTargetComplete = (state: RootState) =>
  !selectNFTTargetError(state);
export const selectNFTIsSendComplete = (state: RootState) =>
  !!selectNFTSignedVAAHex(state);
export const selectNFTIsRedeemComplete = (state: RootState) =>
  !!selectNFTRedeemTx(state);
export const selectNFTShouldLockFields = (state: RootState) =>
  selectNFTIsSending(state) || selectNFTIsSendComplete(state);
export const selectNFTIsRecovery = (state: RootState) => state.nft.isRecovery;
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
export const selectSourceWalletAddress = (state: RootState) =>
  state.transfer.sourceWalletAddress;
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
export const selectTransferTargetAssetWrapper = (state: RootState) =>
  state.transfer.targetAsset;
export const selectTransferTargetAsset = (state: RootState) =>
  state.transfer.targetAsset.data?.address;
export const selectTransferTargetParsedTokenAccount = (state: RootState) =>
  state.transfer.targetParsedTokenAccount;
export const selectTransferTargetBalanceString = (state: RootState) =>
  state.transfer.targetParsedTokenAccount?.uiAmountString || "";
export const selectTransferTransferTx = (state: RootState) =>
  state.transfer.transferTx;
export const selectTransferSignedVAAHex = (state: RootState) =>
  state.transfer.signedVAAHex;
export const selectTransferIsSending = (state: RootState) =>
  state.transfer.isSending;
export const selectTransferIsRedeeming = (state: RootState) =>
  state.transfer.isRedeeming;
export const selectTransferRedeemTx = (state: RootState) =>
  state.transfer.redeemTx;
export const selectTransferIsApproving = (state: RootState) =>
  state.transfer.isApproving;
export const selectTransferSourceError = (
  state: RootState
): string | undefined => {
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
  // no NFT check - NFTs should be blocked by all token pickers
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
  } catch (e: any) {
    if (e?.message) {
      return e.message.substring(0, e.message.indexOf("("));
    }
    return "Invalid amount";
  }
  return undefined;
};
export const selectTransferIsSourceComplete = (state: RootState) =>
  !selectTransferSourceError(state);
export const UNREGISTERED_ERROR_MESSAGE =
  "Target asset unavailable. Is the token registered?";
export const selectTransferTargetError = (state: RootState) => {
  const sourceError = selectTransferSourceError(state);
  if (sourceError) {
    return `Error in source: ${sourceError}`;
  }
  if (!state.transfer.targetChain) {
    return "Select a target chain";
  }
  if (state.transfer.sourceChain === state.transfer.targetChain) {
    return "Select a different target and source";
  }
  if (!selectTransferTargetAsset(state)) {
    return UNREGISTERED_ERROR_MESSAGE;
  }
  if (
    isEVMChain(state.transfer.targetChain) &&
    selectTransferTargetAsset(state) === ethers.constants.AddressZero
  ) {
    return UNREGISTERED_ERROR_MESSAGE;
  }
  if (!state.transfer.targetAddressHex) {
    return "Target account unavailable";
  }
};
export const selectTransferIsTargetComplete = (state: RootState) =>
  !selectTransferTargetError(state);
export const selectTransferIsSendComplete = (state: RootState) =>
  !!selectTransferSignedVAAHex(state);
export const selectTransferIsRedeemComplete = (state: RootState) =>
  !!selectTransferRedeemTx(state);
export const selectTransferShouldLockFields = (state: RootState) =>
  selectTransferIsSending(state) || selectTransferIsSendComplete(state);
export const selectTransferIsRecovery = (state: RootState) =>
  state.transfer.isRecovery;

export const selectSolanaTokenMap = (state: RootState) => {
  return state.tokens.solanaTokenMap;
};

export const selectTerraTokenMap = (state: RootState) => {
  return state.tokens.terraTokenMap;
};

export const selectMarketsMap = (state: RootState) => {
  return state.tokens.marketsMap;
};

export const selectTerraFeeDenom = (state: RootState) => {
  return state.fee.terraFeeDenom;
};
