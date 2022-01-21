import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { TERRA_DEFAULT_FEE_DENOM } from "../utils/consts";

export interface FeeSliceState {
  terraFeeDenom: string;
}

const initialState: FeeSliceState = {
  terraFeeDenom: TERRA_DEFAULT_FEE_DENOM,
};

export const feeSlice = createSlice({
  name: "fee",
  initialState,
  reducers: {
    setTerraFeeDenom: (state, action: PayloadAction<string>) => {
      state.terraFeeDenom = action.payload;
    },
    reset: () => initialState,
  },
});

export const { setTerraFeeDenom, reset } = feeSlice.actions;

export default feeSlice.reducer;
