import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  ETH_TEST_TOKEN_ADDRESS,
  SOL_TEST_TOKEN_ADDRESS,
} from "../utils/consts";

const LAST_STEP = 3;

type Steps = 0 | 1 | 2 | 3;

export interface AttestState {
  activeStep: Steps;
  sourceChain: ChainId;
  sourceAsset: string;
  targetChain: ChainId;
  signedVAAHex: string | undefined;
  isSending: boolean;
  isCreating: boolean;
}

const initialState: AttestState = {
  activeStep: 0,
  sourceChain: CHAIN_ID_SOLANA,
  sourceAsset: SOL_TEST_TOKEN_ADDRESS,
  targetChain: CHAIN_ID_ETH,
  signedVAAHex: undefined,
  isSending: false,
  isCreating: false,
};

export const attestSlice = createSlice({
  name: "attest",
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
      const prevSourceChain = state.sourceChain;
      state.sourceChain = action.payload;
      // TODO: remove or check env - for testing purposes
      if (action.payload === CHAIN_ID_ETH) {
        state.sourceAsset = ETH_TEST_TOKEN_ADDRESS;
      }
      if (action.payload === CHAIN_ID_SOLANA) {
        state.sourceAsset = SOL_TEST_TOKEN_ADDRESS;
      }
      if (state.targetChain === action.payload) {
        state.targetChain = prevSourceChain;
      }
    },
    setSourceAsset: (state, action: PayloadAction<string>) => {
      state.sourceAsset = action.payload;
    },
    setTargetChain: (state, action: PayloadAction<ChainId>) => {
      const prevTargetChain = state.targetChain;
      state.targetChain = action.payload;
      if (state.sourceChain === action.payload) {
        state.sourceChain = prevTargetChain;
        state.activeStep = 0;
        // TODO: remove or check env - for testing purposes
        if (state.targetChain === CHAIN_ID_ETH) {
          state.sourceAsset = ETH_TEST_TOKEN_ADDRESS;
        }
        if (state.targetChain === CHAIN_ID_SOLANA) {
          state.sourceAsset = SOL_TEST_TOKEN_ADDRESS;
        }
      }
    },
    setSignedVAAHex: (state, action: PayloadAction<string>) => {
      state.signedVAAHex = action.payload;
      state.isSending = false;
      state.activeStep = 3;
    },
    setIsSending: (state, action: PayloadAction<boolean>) => {
      state.isSending = action.payload;
    },
    setIsCreating: (state, action: PayloadAction<boolean>) => {
      state.isCreating = action.payload;
    },
  },
});

export const {
  incrementStep,
  decrementStep,
  setStep,
  setSourceChain,
  setSourceAsset,
  setTargetChain,
  setSignedVAAHex,
  setIsSending,
  setIsCreating,
} = attestSlice.actions;

export default attestSlice.reducer;
