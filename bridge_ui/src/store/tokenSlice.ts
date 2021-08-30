import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { TokenInfo } from "@solana/spl-token-registry";
import {
  DataWrapper,
  errorDataWrapper,
  fetchDataWrapper,
  getEmptyDataWrapper,
  receiveDataWrapper,
} from "./helpers";

export interface TokenMetadataState {
  solanaTokenMap: DataWrapper<TokenInfo[]>;
}

const initialState: TokenMetadataState = {
  solanaTokenMap: getEmptyDataWrapper(),
};

export const tokenSlice = createSlice({
  name: "tokenInfos",
  initialState,
  reducers: {
    receiveSolanaTokenMap: (state, action: PayloadAction<TokenInfo[]>) => {
      state.solanaTokenMap = receiveDataWrapper(action.payload);
    },
    fetchSolanaTokenMap: (state) => {
      state.solanaTokenMap = fetchDataWrapper();
    },
    errorSolanaTokenMap: (state, action: PayloadAction<string>) => {
      state.solanaTokenMap = errorDataWrapper(action.payload);
    },
    reset: () => initialState,
  },
});

export const {
  receiveSolanaTokenMap,
  fetchSolanaTokenMap,
  errorSolanaTokenMap,
  reset,
} = tokenSlice.actions;

export default tokenSlice.reducer;
