import {
  ChainId,
  CHAIN_ID_ACALA,
  CHAIN_ID_KARURA,
} from "@certusone/wormhole-sdk";
import axios from "axios";
import { parseUnits } from "ethers/lib/utils";
import { useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import {
  selectAcalaRelayerInfo,
  selectTransferSourceAsset,
  selectTransferSourceParsedTokenAccount,
} from "../store/selectors";
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
  transferAmount: string
) => {
  const dispatch = useDispatch();
  const sourceAsset = useSelector(selectTransferSourceAsset);
  const sourceParsedTokenAccount = useSelector(
    selectTransferSourceParsedTokenAccount
  );
  const decimals = sourceParsedTokenAccount?.decimals;
  useEffect(() => {
    let cancelled = false;
    if (
      !ACALA_RELAYER_URL ||
      !targetChain ||
      (targetChain !== CHAIN_ID_ACALA && targetChain !== CHAIN_ID_KARURA) ||
      !transferAmount ||
      !sourceAsset
    ) {
      dispatch(setAcalaRelayerInfo());
      return;
    }
    dispatch(fetchAcalaRelayerInfo());
    (async () => {
      try {
        const amountParsed = parseUnits(transferAmount, decimals).toString();

        const result = await axios.get(ACALA_SHOULD_RELAY_URL, {
          params: {
            targetChain,
            sourceAsset,
            amount: amountParsed,
          },
        });

        console.log("check should relay: ", {
          targetChain,
          sourceAsset,
          amount: amountParsed,
          result: result.data?.shouldRelay,
        });
        if (!cancelled) {
          dispatch(receiveAcalaRelayerInfo(result.data));
        }
      } catch (e) {
        if (!cancelled) {
          dispatch(
            errorAcalaRelayerInfo("Failed to retrieve the Acala relayer info.")
          );
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [targetChain, transferAmount, sourceAsset, decimals, dispatch]);
  return useSelector(selectAcalaRelayerInfo);
};
