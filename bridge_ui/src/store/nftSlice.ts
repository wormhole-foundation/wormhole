import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { StateSafeWormholeWrappedInfo } from "../hooks/useCheckIfWormholeWrapped";
import { ForeignAssetInfo } from "../hooks/useFetchForeignAsset";
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
  nftName?: string;
  description?: string;
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
  targetAsset: DataWrapper<ForeignAssetInfo>;
  transferTx: Transaction | undefined;
  signedVAAHex: string | undefined;
  isSending: boolean;
  isRedeeming: boolean;
  redeemTx: Transaction | undefined;
  isRecovery: boolean;
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
  targetAsset: getEmptyDataWrapper(),
  transferTx: undefined,
  signedVAAHex: undefined,
  isSending: false,
  isRedeeming: false,
  redeemTx: undefined,
  isRecovery: false,
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
      // clear targetAsset so that components that fire before useFetchTargetAsset don't get stale data
      state.targetAsset = getEmptyDataWrapper();
      state.targetAddressHex = undefined;
      state.isSourceAssetWormholeWrapped = undefined;
      state.originChain = undefined;
      state.originAsset = undefined;
      state.originTokenId = undefined;
      if (state.targetChain === action.payload) {
        state.targetChain = prevSourceChain;
      }
    },
    setSourceWormholeWrappedInfo: (
      state,
      action: PayloadAction<StateSafeWormholeWrappedInfo>
    ) => {
      state.isSourceAssetWormholeWrapped = action.payload.isWrapped;
      state.originChain = action.payload.chainId;
      state.originAsset = action.payload.assetAddress;
      state.originTokenId = action.payload.tokenId;
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
      // clear targetAsset so that components that fire before useFetchTargetAsset don't get stale data
      state.targetAsset = getEmptyDataWrapper();
      state.targetAddressHex = undefined;
      state.isSourceAssetWormholeWrapped = undefined;
      state.originChain = undefined;
      state.originAsset = undefined;
      state.originTokenId = undefined;
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
      state.targetAsset = getEmptyDataWrapper();
      if (state.sourceChain === action.payload) {
        state.sourceChain = prevTargetChain;
        state.activeStep = 0;
        state.sourceParsedTokenAccount = undefined;
        state.isSourceAssetWormholeWrapped = undefined;
        state.originChain = undefined;
        state.originAsset = undefined;
        state.originTokenId = undefined;
        state.sourceParsedTokenAccounts = getEmptyDataWrapper();
      }
    },
    setTargetAddressHex: (state, action: PayloadAction<string | undefined>) => {
      state.targetAddressHex = action.payload;
    },
    setTargetAsset: (
      state,
      action: PayloadAction<DataWrapper<ForeignAssetInfo>>
    ) => {
      state.targetAsset = action.payload;
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
    setRecoveryVaa: (
      state,
      action: PayloadAction<{
        vaa: any;
        parsedPayload: {
          targetChain: ChainId;
          targetAddress: string;
          originChain: ChainId;
          originAddress: string; //TODO maximum amount of fields
        };
      }>
    ) => {
      const prevTargetChain = state.targetChain;
      state.signedVAAHex = action.payload.vaa;
      state.targetChain = action.payload.parsedPayload.targetChain;
      if (state.sourceChain === action.payload.parsedPayload.targetChain) {
        state.sourceChain = prevTargetChain;
      }
      state.sourceParsedTokenAccount = undefined;
      state.sourceParsedTokenAccounts = getEmptyDataWrapper();
      state.targetAsset = getEmptyDataWrapper();
      state.isSourceAssetWormholeWrapped = undefined;
      state.targetAddressHex = action.payload.parsedPayload.targetAddress;
      state.originChain = action.payload.parsedPayload.originChain;
      state.originAsset = action.payload.parsedPayload.originAddress;
      state.originTokenId = undefined;
      state.activeStep = 3;
      state.isRecovery = true;
    },
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
  setTransferTx,
  setSignedVAAHex,
  setIsSending,
  setIsRedeeming,
  setRedeemTx,
  reset,
  setRecoveryVaa,
} = nftSlice.actions;

export default nftSlice.reducer;
