import { parseUnits } from "ethers/lib/utils";
import { RootState } from ".";
import { CHAIN_ID_SOLANA } from "../utils/consts";

export const selectActiveStep = (state: RootState) => state.transfer.activeStep;
export const selectSourceChain = (state: RootState) =>
  state.transfer.sourceChain;
export const selectSourceAsset = (state: RootState) =>
  state.transfer.sourceAsset;
export const selectSourceParsedTokenAccount = (state: RootState) =>
  state.transfer.sourceParsedTokenAccount;
export const selectSourceBalanceString = (state: RootState) =>
  state.transfer.sourceParsedTokenAccount?.uiAmountString || "";
export const selectAmount = (state: RootState) => state.transfer.amount;
export const selectTargetChain = (state: RootState) =>
  state.transfer.targetChain;
export const selectSignedVAAHex = (state: RootState) =>
  state.transfer.signedVAAHex;
export const selectIsSending = (state: RootState) => state.transfer.isSending;
export const selectIsRedeeming = (state: RootState) =>
  state.transfer.isRedeeming;

// safety checks
// TODO: could make this return a string with a user informative message
export const selectIsSourceComplete = (state: RootState) =>
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
export const selectIsTargetComplete = (state: RootState) =>
  selectIsSourceComplete(state) && !!state.transfer.targetChain;
export const selectIsSendComplete = (state: RootState) =>
  !!selectSignedVAAHex(state);
export const selectShouldLockFields = (state: RootState) =>
  selectIsSending(state) || selectIsSendComplete(state);
