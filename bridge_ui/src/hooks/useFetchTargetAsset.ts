import { CHAIN_ID_TERRA, CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import {
  selectTransferIsSourceAssetWormholeWrapped,
  selectTransferOriginAsset,
  selectTransferOriginChain,
  selectTransferSourceAsset,
  selectTransferSourceChain,
  selectTransferTargetChain,
} from "../store/selectors";
import { setTargetAsset } from "../store/transferSlice";
import {
  getForeignAssetEth,
  getForeignAssetSol,
  getForeignAssetTerra,
} from "../utils/getForeignAsset";

function useFetchTargetAsset() {
  const dispatch = useDispatch();
  const sourceChain = useSelector(selectTransferSourceChain);
  const sourceAsset = useSelector(selectTransferSourceAsset);
  const isSourceAssetWormholeWrapped = useSelector(
    selectTransferIsSourceAssetWormholeWrapped
  );
  const originChain = useSelector(selectTransferOriginChain);
  const originAsset = useSelector(selectTransferOriginAsset);
  const targetChain = useSelector(selectTransferTargetChain);
  const { provider } = useEthereumProvider();
  // TODO: this may not cover wrapped to wrapped, should always use origin?
  useEffect(() => {
    if (isSourceAssetWormholeWrapped && originChain === targetChain) {
      dispatch(setTargetAsset(originAsset));
      return;
    }
    // TODO: loading state, error state
    dispatch(setTargetAsset(undefined));
    let cancelled = false;
    (async () => {
      if (targetChain === CHAIN_ID_ETH && provider) {
        const asset = await getForeignAssetEth(
          provider,
          sourceChain,
          sourceAsset
        );
        if (!cancelled) {
          dispatch(setTargetAsset(asset));
        }
      }
      if (targetChain === CHAIN_ID_SOLANA) {
        try {
          const asset = await getForeignAssetSol(sourceChain, sourceAsset);
          if (!cancelled) {
            dispatch(setTargetAsset(asset));
          }
        } catch (e) {
          if (!cancelled) {
            // TODO: warning for this
          }
        }
      }
      if (targetChain === CHAIN_ID_TERRA) {
        try {
          const asset = await getForeignAssetTerra(sourceChain, sourceAsset);
          if (!cancelled) {
            console.log("terra target asset", asset);
            dispatch(setTargetAsset(asset));
          }
        } catch (e) {
          if (!cancelled) {
            // TODO: warning for this
          }
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [
    dispatch,
    isSourceAssetWormholeWrapped,
    originChain,
    originAsset,
    targetChain,
    sourceChain,
    sourceAsset,
    provider,
  ]);
}

export default useFetchTargetAsset;
