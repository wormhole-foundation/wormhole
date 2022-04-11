import {
  ChainId,
  CHAIN_ID_ACALA,
  CHAIN_ID_KARURA,
} from "@certusone/wormhole-sdk";
import axios from "axios";
import { useEffect, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import {
  DataWrapper,
  errorDataWrapper,
  fetchDataWrapper,
  getEmptyDataWrapper,
  receiveDataWrapper,
} from "../store/helpers";
import { selectAcalaRelayerInfo } from "../store/selectors";
import {
  errorAcalaRelayerInfo,
  fetchAcalaRelayerInfo,
  receiveAcalaRelayerInfo,
  setAcalaRelayerInfo,
} from "../store/transferSlice";
import { ACALA_RELAYER_URL, ACALA_SHOULD_RELAY_URL } from "../utils/consts";

export interface AcalaRelayerInfo {
  shouldRelay: boolean;
  msg: string;
}

export const useAcalaRelayerInfo = (
  targetChain: ChainId,
  vaaNormalizedAmount: string | undefined,
  originAsset: string | undefined,
  useStore: boolean = true
) => {
  // within flow, update the store
  const dispatch = useDispatch();
  // within recover, use internal state
  const [state, setState] = useState<DataWrapper<AcalaRelayerInfo>>(
    getEmptyDataWrapper()
  );
  useEffect(() => {
    let cancelled = false;
    if (
      !ACALA_RELAYER_URL ||
      !targetChain ||
      (targetChain !== CHAIN_ID_ACALA && targetChain !== CHAIN_ID_KARURA) ||
      !vaaNormalizedAmount ||
      !originAsset
    ) {
      useStore
        ? dispatch(setAcalaRelayerInfo())
        : setState(getEmptyDataWrapper());
      return;
    }
    useStore ? dispatch(fetchAcalaRelayerInfo()) : setState(fetchDataWrapper());
    (async () => {
      try {
        const result = await axios.get(ACALA_SHOULD_RELAY_URL, {
          params: {
            targetChain,
            originAsset,
            amount: vaaNormalizedAmount,
          },
        });

        console.log("check should relay: ", {
          targetChain,
          originAsset,
          amount: vaaNormalizedAmount,
          result: result.data?.shouldRelay,
        });
        if (!cancelled) {
          useStore
            ? dispatch(receiveAcalaRelayerInfo(result.data))
            : setState(receiveDataWrapper(result.data));
        }
      } catch (e) {
        if (!cancelled) {
          useStore
            ? dispatch(
                errorAcalaRelayerInfo(
                  "Failed to retrieve the Acala relayer info."
                )
              )
            : setState(
                errorDataWrapper("Failed to retrieve the Acala relayer info.")
              );
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [targetChain, vaaNormalizedAmount, originAsset, dispatch, useStore]);
  const acalaRelayerInfoFromStore = useSelector(selectAcalaRelayerInfo);
  return useStore ? acalaRelayerInfoFromStore : state;
};
