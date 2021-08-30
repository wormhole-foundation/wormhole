import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
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
  getOriginalAssetTerra,
} from "../utils/getOriginalAsset";

// Check if the tokens in the configured source chain/address are wrapped
// tokens. Wrapped tokens are tokens that are non-native, I.E, are locked up on
// a different chain than this one.
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
      if (sourceChain === CHAIN_ID_ETH && provider && sourceAsset) {
        const wrappedInfo = await getOriginalAssetEth(provider, sourceAsset);
        if (!cancelled) {
          dispatch(setSourceWormholeWrappedInfo(wrappedInfo));
        }
      }
      if (sourceChain === CHAIN_ID_SOLANA && sourceAsset) {
        try {
          const wrappedInfo = await getOriginalAssetSol(sourceAsset);
          if (!cancelled) {
            dispatch(setSourceWormholeWrappedInfo(wrappedInfo));
          }
        } catch (e) {}
      }
      if (sourceChain === CHAIN_ID_TERRA && sourceAsset) {
        try {
          const wrappedInfo = await getOriginalAssetTerra(sourceAsset);
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
