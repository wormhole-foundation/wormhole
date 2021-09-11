import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { TokenInfo } from "@solana/spl-token-registry";
import { TerraTokenMap } from "../hooks/useTerraTokenMap";
import {
  DataWrapper,
  errorDataWrapper,
  fetchDataWrapper,
  getEmptyDataWrapper,
  receiveDataWrapper,
} from "./helpers";

export interface TokenMetadataState {
  solanaTokenMap: DataWrapper<TokenInfo[]>;
  terraTokenMap: DataWrapper<TerraTokenMap>; //TODO make a decent type for this.
}

const initialState: TokenMetadataState = {
  solanaTokenMap: getEmptyDataWrapper(),
  terraTokenMap: getEmptyDataWrapper(),
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

    receiveTerraTokenMap: (state, action: PayloadAction<TerraTokenMap>) => {
      state.terraTokenMap = receiveDataWrapper(action.payload);
    },
    fetchTerraTokenMap: (state) => {
      state.terraTokenMap = fetchDataWrapper();
    },
    errorTerraTokenMap: (state, action: PayloadAction<string>) => {
      state.terraTokenMap = errorDataWrapper(action.payload);
    },

    reset: () => initialState,
  },
});

export const {
  receiveSolanaTokenMap,
  fetchSolanaTokenMap,
  errorSolanaTokenMap,
  receiveTerraTokenMap,
  fetchTerraTokenMap,
  errorTerraTokenMap,
  reset,
} = tokenSlice.actions;

export default tokenSlice.reducer;
