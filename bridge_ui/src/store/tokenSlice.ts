import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { TokenInfo } from "@solana/spl-token-registry";
import { MarketsMap } from "../hooks/useMarketsMap";
import { RelayerTokenInfo } from "../hooks/useRelayersAvailable";
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
  marketsMap: DataWrapper<MarketsMap>;
  relayerTokenInfo: DataWrapper<RelayerTokenInfo>;
}

const initialState: TokenMetadataState = {
  solanaTokenMap: getEmptyDataWrapper(),
  terraTokenMap: getEmptyDataWrapper(),
  marketsMap: getEmptyDataWrapper(),
  relayerTokenInfo: getEmptyDataWrapper(),
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

    receiveMarketsMap: (state, action: PayloadAction<MarketsMap>) => {
      state.marketsMap = receiveDataWrapper(action.payload);
    },
    fetchMarketsMap: (state) => {
      state.marketsMap = fetchDataWrapper();
    },
    errorMarketsMap: (state, action: PayloadAction<string>) => {
      state.marketsMap = errorDataWrapper(action.payload);
    },

    receiveRelayerTokenInfo: (
      state,
      action: PayloadAction<RelayerTokenInfo>
    ) => {
      state.relayerTokenInfo = receiveDataWrapper(action.payload);
    },
    fetchRelayerTokenInfo: (state) => {
      state.relayerTokenInfo = fetchDataWrapper();
    },
    errorRelayerTokenInfo: (state, action: PayloadAction<string>) => {
      state.relayerTokenInfo = errorDataWrapper(action.payload);
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
  receiveMarketsMap,
  fetchMarketsMap,
  errorMarketsMap,
  receiveRelayerTokenInfo,
  fetchRelayerTokenInfo,
  errorRelayerTokenInfo,
  reset,
} = tokenSlice.actions;

export default tokenSlice.reducer;
