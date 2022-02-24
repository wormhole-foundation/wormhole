import { configureStore } from "@reduxjs/toolkit";
import attestReducer from "./attestSlice";
import nftReducer from "./nftSlice";
import transferReducer from "./transferSlice";
import tokenReducer from "./tokenSlice";
import feeReducer from "./feeSlice";

export const store = configureStore({
  reducer: {
    attest: attestReducer,
    nft: nftReducer,
    transfer: transferReducer,
    tokens: tokenReducer,
    fee: feeReducer,
  },
});

// Infer the `RootState` and `AppDispatch` types from the store itself
export type RootState = ReturnType<typeof store.getState>;
// Inferred type: {posts: PostsState, comments: CommentsState, users: UsersState}
export type AppDispatch = typeof store.dispatch;
