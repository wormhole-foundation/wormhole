import { parseUnits } from "ethers/lib/utils";
import { RootState } from ".";
import { CHAIN_ID_SOLANA } from "../utils/consts";

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

// safety checks
// TODO: could make this return a string with a user informative message
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
export const selectTransferSourceAsset = (state: RootState) =>
  state.transfer.sourceAsset;
export const selectTransferSourceParsedTokenAccount = (state: RootState) =>
  state.transfer.sourceParsedTokenAccount;
export const selectTransferSourceBalanceString = (state: RootState) =>
  state.transfer.sourceParsedTokenAccount?.uiAmountString || "";
export const selectTransferAmount = (state: RootState) => state.transfer.amount;
export const selectTransferTargetChain = (state: RootState) =>
  state.transfer.targetChain;
export const selectTransferSignedVAAHex = (state: RootState) =>
  state.transfer.signedVAAHex;
export const selectTransferIsSending = (state: RootState) =>
  state.transfer.isSending;
export const selectTransferIsRedeeming = (state: RootState) =>
  state.transfer.isRedeeming;

// safety checks
// TODO: could make this return a string with a user informative message
export const selectTransferIsSourceComplete = (state: RootState) =>
  !!state.transfer.sourceChain &&
  !!state.transfer.sourceAsset &&
  !!state.transfer.sourceParsedTokenAccount &&
  !!state.transfer.amount &&
  (state.transfer.sourceChain !== CHAIN_ID_SOLANA ||
    !!state.transfer.sourceParsedTokenAccount.publicKey) &&
  !!state.transfer.sourceParsedTokenAccount.uiAmountString &&
  // TODO: make safe with too many decimals
  parseUnits(
    state.transfer.amount,
    state.transfer.sourceParsedTokenAccount.decimals
  ).lte(
    parseUnits(
      state.transfer.sourceParsedTokenAccount.uiAmountString,
      state.transfer.sourceParsedTokenAccount.decimals
    )
  );
// TODO: check wrapped asset exists or is native transfer
export const selectTransferIsTargetComplete = (state: RootState) =>
  selectTransferIsSourceComplete(state) && !!state.transfer.targetChain;
export const selectTransferIsSendComplete = (state: RootState) =>
  !!selectTransferSignedVAAHex(state);
export const selectTransferShouldLockFields = (state: RootState) =>
  selectTransferIsSending(state) || selectTransferIsSendComplete(state);
