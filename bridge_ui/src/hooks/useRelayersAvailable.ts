import { ChainId } from "@certusone/wormhole-sdk";
import { Dispatch } from "@reduxjs/toolkit";
import axios from "axios";
import { useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import { DataWrapper } from "../store/helpers";
import { selectRelayerTokenInfo } from "../store/selectors";
import {
  errorRelayerTokenInfo,
  fetchRelayerTokenInfo,
  receiveRelayerTokenInfo,
} from "../store/tokenSlice";
import { RELAYER_INFO_URL } from "../utils/consts";

export type RelayToken = {
  chainId?: ChainId;
  address?: string;
  coingeckoId?: string;
};
export type Relayer = {
  name?: string;
  url?: string;
};
export type FeeScheduleEntryFlat = {
  type: "flat";
  feeUsd: number;
};
export type FeeScheduleEntryPercent = {
  type: "percent";
  feePercent: number;
  gasEstimate: number;
};
export type FeeSchedule = {
  // ChainId as a string
  [key: string]: FeeScheduleEntryFlat | FeeScheduleEntryPercent;
};
export type RelayerTokenInfo = {
  supportedTokens?: RelayToken[];
  relayers?: Relayer[];
  feeSchedule?: FeeSchedule;
};

const useRelayersAvailable = (
  shouldFire: boolean
): DataWrapper<RelayerTokenInfo> => {
  const relayerTokenInfo = useSelector(selectRelayerTokenInfo);
  console.log("relayerTokenInfo", relayerTokenInfo);
  const dispatch = useDispatch();
  const internalShouldFire =
    shouldFire &&
    (relayerTokenInfo.data === undefined ||
      (relayerTokenInfo.data === null && !relayerTokenInfo.isFetching));

  useEffect(() => {
    if (internalShouldFire) {
      getRelayersAvailable(dispatch);
    }
  }, [internalShouldFire, dispatch]);

  return relayerTokenInfo;
};

const getRelayersAvailable = (dispatch: Dispatch) => {
  dispatch(fetchRelayerTokenInfo());
  axios.get(RELAYER_INFO_URL).then(
    (response) => {
      dispatch(receiveRelayerTokenInfo(response.data as RelayerTokenInfo));
    },
    (error) => {
      dispatch(
        errorRelayerTokenInfo("Failed to retrieve the relayer token info.")
      );
    }
  );
};

export default useRelayersAvailable;
