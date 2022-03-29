import { ChainId } from "@certusone/wormhole-sdk";
import { Dispatch } from "@reduxjs/toolkit";
import axios from "axios";
import { useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import { DataWrapper } from "../store/helpers";
import { selectRelayerTokenInfo } from "../store/selectors";
import {
  errorRelayerTokenInfo,
  fetchTerraTokenMap,
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
export type RelayerTokenInfo = {
  supportedTokens?: RelayToken[];
  relayers?: Relayer[];
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
  dispatch(fetchTerraTokenMap());
  axios.get(RELAYER_INFO_URL).then(
    (response) => {
      dispatch(receiveRelayerTokenInfo(response.data as RelayerTokenInfo));
    },
    (error) => {
      dispatch(
        errorRelayerTokenInfo("Failed to retrieve the Terra Token List.")
      );
    }
  );
};

export default useRelayersAvailable;
