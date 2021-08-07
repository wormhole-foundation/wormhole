import { RootState } from ".";

export const selectActiveStep = (state: RootState) => state.transfer.activeStep;
export const selectSourceChain = (state: RootState) =>
  state.transfer.sourceChain;
export const selectTargetChain = (state: RootState) =>
  state.transfer.targetChain;
export const selectSignedVAA = (state: RootState) => state.transfer.signedVAA; //TODO: deserialize
