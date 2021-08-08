import { RootState } from ".";

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
