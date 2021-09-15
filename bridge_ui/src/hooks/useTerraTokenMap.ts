import { Dispatch } from "@reduxjs/toolkit";
import axios from "axios";
import { useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import { DataWrapper } from "../store/helpers";
import { selectTerraTokenMap } from "../store/selectors";
import {
  errorTerraTokenMap,
  fetchTerraTokenMap,
  receiveTerraTokenMap,
} from "../store/tokenSlice";
import { TERRA_TOKEN_METADATA_URL } from "../utils/consts";

export type TerraTokenMetadata = {
  protocol: string;
  symbol: string;
  token: string;
  icon: string;
};

export type TerraTokenMap = {
  mainnet: {
    [address: string]: TerraTokenMetadata;
  };
  testnet: {
    [address: string]: TerraTokenMetadata;
  };
};

const useTerraTokenMap = (): DataWrapper<TerraTokenMap> => {
  const terraTokenMap = useSelector(selectTerraTokenMap);
  const dispatch = useDispatch();
  const shouldFire =
    terraTokenMap.data === undefined ||
    (terraTokenMap.data === null && !terraTokenMap.isFetching);

  useEffect(() => {
    if (shouldFire) {
      getTerraTokenMap(dispatch);
    }
  }, [shouldFire, dispatch]);

  return terraTokenMap;
};

const getTerraTokenMap = (dispatch: Dispatch) => {
  dispatch(fetchTerraTokenMap());
  axios.get(TERRA_TOKEN_METADATA_URL).then(
    (response) => {
      dispatch(receiveTerraTokenMap(response.data as TerraTokenMap));
    },
    (error) => {
      dispatch(errorTerraTokenMap("Failed to retrieve the Terra Token List."));
    }
  );
};

export default useTerraTokenMap;
