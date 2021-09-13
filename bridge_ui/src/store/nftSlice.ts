import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { StateSafeWormholeWrappedInfo } from "../hooks/useCheckIfWormholeWrapped";
import {
  DataWrapper,
  errorDataWrapper,
  fetchDataWrapper,
  getEmptyDataWrapper,
  receiveDataWrapper,
} from "./helpers";
import { ParsedTokenAccount, Transaction } from "./transferSlice";

const LAST_STEP = 3;

type Steps = 0 | 1 | 2 | 3;

// these all are optional so NFT could share TokenSelectors
export interface NFTParsedTokenAccount extends ParsedTokenAccount {
  tokenId?: string;
  uri?: string;
  animation_url?: string | null;
  external_url?: string | null;
  image?: string;
  image_256?: string;
  name?: string;
}

export interface NFTState {
  activeStep: Steps;
  sourceChain: ChainId;
  isSourceAssetWormholeWrapped: boolean | undefined;
  originChain: ChainId | undefined;
  originAsset: string | undefined;
  originTokenId: string | undefined;
  sourceWalletAddress: string | undefined;
  sourceParsedTokenAccount: NFTParsedTokenAccount | undefined;
  sourceParsedTokenAccounts: DataWrapper<NFTParsedTokenAccount[]>;
  targetChain: ChainId;
  targetAddressHex: string | undefined;
  targetAsset: string | null | undefined;
  targetParsedTokenAccount: NFTParsedTokenAccount | undefined;
  transferTx: Transaction | undefined;
  signedVAAHex: string | undefined;
  isSending: boolean;
  isRedeeming: boolean;
  redeemTx: Transaction | undefined;
}

const initialState: NFTState = {
  activeStep: 0,
  sourceChain: CHAIN_ID_SOLANA,
  isSourceAssetWormholeWrapped: false,
  sourceWalletAddress: undefined,
  sourceParsedTokenAccount: undefined,
  sourceParsedTokenAccounts: getEmptyDataWrapper(),
  originChain: undefined,
  originAsset: undefined,
  originTokenId: undefined,
  targetChain: CHAIN_ID_ETH,
  targetAddressHex: undefined,
  targetAsset: undefined,
  targetParsedTokenAccount: undefined,
  transferTx: undefined,
  signedVAAHex: undefined,
  isSending: false,
  isRedeeming: false,
  redeemTx: undefined,
};

export const nftSlice = createSlice({
  name: "nft",
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
      state.sourceParsedTokenAccount = undefined;
      state.sourceParsedTokenAccounts = getEmptyDataWrapper();
      if (state.targetChain === action.payload) {
        state.targetChain = prevSourceChain;
        state.targetAddressHex = undefined;
        // clear targetAsset so that components that fire before useFetchTargetAsset don't get stale data
        state.targetAsset = undefined;
        state.targetParsedTokenAccount = undefined;
      }
    },
    setSourceWormholeWrappedInfo: (
      state,
      action: PayloadAction<StateSafeWormholeWrappedInfo | undefined>
    ) => {
      if (action.payload) {
        state.isSourceAssetWormholeWrapped = action.payload.isWrapped;
        state.originChain = action.payload.chainId;
        state.originAsset = action.payload.assetAddress;
        state.originTokenId = action.payload.tokenId;
      } else {
        state.isSourceAssetWormholeWrapped = undefined;
        state.originChain = undefined;
        state.originAsset = undefined;
        state.originTokenId = undefined;
      }
    },
    setSourceWalletAddress: (
      state,
      action: PayloadAction<string | undefined>
    ) => {
      state.sourceWalletAddress = action.payload;
    },
    setSourceParsedTokenAccount: (
      state,
      action: PayloadAction<NFTParsedTokenAccount | undefined>
    ) => {
      state.sourceParsedTokenAccount = action.payload;
    },
    setSourceParsedTokenAccounts: (
      state,
      action: PayloadAction<NFTParsedTokenAccount[] | undefined>
    ) => {
      state.sourceParsedTokenAccounts = action.payload
        ? receiveDataWrapper(action.payload)
        : getEmptyDataWrapper();
    },
    fetchSourceParsedTokenAccounts: (state) => {
      state.sourceParsedTokenAccounts = fetchDataWrapper();
    },
    errorSourceParsedTokenAccounts: (
      state,
      action: PayloadAction<string | undefined>
    ) => {
      state.sourceParsedTokenAccounts = errorDataWrapper(
        action.payload || "An unknown error occurred."
      );
    },
    receiveSourceParsedTokenAccounts: (
      state,
      action: PayloadAction<NFTParsedTokenAccount[]>
    ) => {
      state.sourceParsedTokenAccounts = receiveDataWrapper(action.payload);
    },
    setTargetChain: (state, action: PayloadAction<ChainId>) => {
      const prevTargetChain = state.targetChain;
      state.targetChain = action.payload;
      state.targetAddressHex = undefined;
      // clear targetAsset so that components that fire before useFetchTargetAsset don't get stale data
      state.targetAsset = undefined;
      state.targetParsedTokenAccount = undefined;
      if (state.sourceChain === action.payload) {
        state.sourceChain = prevTargetChain;
        state.activeStep = 0;
        state.sourceParsedTokenAccount = undefined;
        state.sourceParsedTokenAccounts = getEmptyDataWrapper();
      }
    },
    setTargetAddressHex: (state, action: PayloadAction<string | undefined>) => {
      state.targetAddressHex = action.payload;
    },
    setTargetAsset: (
      state,
      action: PayloadAction<string | null | undefined>
    ) => {
      state.targetAsset = action.payload;
    },
    setTargetParsedTokenAccount: (
      state,
      action: PayloadAction<NFTParsedTokenAccount | undefined>
    ) => {
      state.targetParsedTokenAccount = action.payload;
    },
    setTransferTx: (state, action: PayloadAction<Transaction>) => {
      state.transferTx = action.payload;
    },
    setSignedVAAHex: (state, action: PayloadAction<string>) => {
      state.signedVAAHex = action.payload;
      state.isSending = false;
      state.activeStep = 3;
    },
    setIsSending: (state, action: PayloadAction<boolean>) => {
      state.isSending = action.payload;
    },
    setIsRedeeming: (state, action: PayloadAction<boolean>) => {
      state.isRedeeming = action.payload;
    },
    setRedeemTx: (state, action: PayloadAction<Transaction>) => {
      state.redeemTx = action.payload;
      state.isRedeeming = false;
    },
    reset: (state) => ({
      ...initialState,
      sourceChain: state.sourceChain,
      targetChain: state.targetChain,
    }),
  },
});

export const {
  incrementStep,
  decrementStep,
  setStep,
  setSourceChain,
  setSourceWormholeWrappedInfo,
  setSourceWalletAddress,
  setSourceParsedTokenAccount,
  setSourceParsedTokenAccounts,
  receiveSourceParsedTokenAccounts,
  errorSourceParsedTokenAccounts,
  fetchSourceParsedTokenAccounts,
  setTargetChain,
  setTargetAddressHex,
  setTargetAsset,
  setTargetParsedTokenAccount,
  setTransferTx,
  setSignedVAAHex,
  setIsSending,
  setIsRedeeming,
  setRedeemTx,
  reset,
} = nftSlice.actions;

export default nftSlice.reducer;
