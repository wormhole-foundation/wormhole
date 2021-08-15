import { CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import {
  selectTransferSourceAsset,
  selectTransferSourceChain,
} from "../store/selectors";
import { setSourceWormholeWrappedInfo } from "../store/transferSlice";
import {
  getOriginalAssetEth,
  getOriginalAssetSol,
} from "../utils/getOriginalAsset";

function useCheckIfWormholeWrapped() {
  const dispatch = useDispatch();
  const sourceChain = useSelector(selectTransferSourceChain);
  const sourceAsset = useSelector(selectTransferSourceAsset);
  const { provider } = useEthereumProvider();
  useEffect(() => {
    // TODO: loading state, error state
    dispatch(setSourceWormholeWrappedInfo(undefined));
    let cancelled = false;
    (async () => {
      if (sourceChain === CHAIN_ID_ETH && provider) {
        const wrappedInfo = await getOriginalAssetEth(provider, sourceAsset);
        if (!cancelled) {
          dispatch(setSourceWormholeWrappedInfo(wrappedInfo));
        }
      } else if (sourceChain === CHAIN_ID_SOLANA) {
        try {
          const wrappedInfo = await getOriginalAssetSol(sourceAsset);
          if (!cancelled) {
            dispatch(setSourceWormholeWrappedInfo(wrappedInfo));
          }
        } catch (e) {}
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [dispatch, sourceChain, sourceAsset, provider]);
}

export default useCheckIfWormholeWrapped;
