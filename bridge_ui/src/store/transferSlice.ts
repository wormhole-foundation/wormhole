import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { ChainId, CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "../utils/consts";

const LAST_STEP = 3;

type Steps = 0 | 1 | 2 | 3;

export interface TransferState {
  activeStep: Steps;
  sourceChain: ChainId;
  targetChain: ChainId;
  signedVAA: Uint8Array | undefined;
}

const initialState: TransferState = {
  activeStep: 0,
  sourceChain: CHAIN_ID_SOLANA,
  targetChain: CHAIN_ID_ETH,
  signedVAA: undefined,
};

export const transferSlice = createSlice({
  name: "transfer",
  initialState,
  reducers: {
    incrementStep: (state) => {
      if (state.activeStep < LAST_STEP) state.activeStep++;
    },
    decrementStep: (state) => {
      if (state.activeStep > 0) state.activeStep--;
    },
    setStep: (state, action: PayloadAction<Steps>) => {
      state.activeStep = action.payload;
    },
    setSourceChain: (state, action: PayloadAction<ChainId>) => {
      state.sourceChain = action.payload;
    },
    setTargetChain: (state, action: PayloadAction<ChainId>) => {
      state.targetChain = action.payload;
    },
    setSignedVAA: (state, action: PayloadAction<Uint8Array>) => {
      state.signedVAA = action.payload; //TODO: serialize
      state.activeStep = 3;
    },
  },
});

export const {
  incrementStep,
  decrementStep,
  setStep,
  setSourceChain,
  setTargetChain,
  setSignedVAA,
} = transferSlice.actions;

export default transferSlice.reducer;
