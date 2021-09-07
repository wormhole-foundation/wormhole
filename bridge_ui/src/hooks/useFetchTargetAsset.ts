import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import {
  selectTransferIsSourceAssetWormholeWrapped,
  selectTransferOriginAsset,
  selectTransferOriginChain,
  selectTransferTargetChain,
} from "../store/selectors";
import { setTargetAsset } from "../store/transferSlice";
import { hexToNativeString } from "../utils/array";
import {
  getForeignAssetEth,
  getForeignAssetSol,
  getForeignAssetTerra,
} from "../utils/getForeignAsset";

function useFetchTargetAsset() {
  const dispatch = useDispatch();
  const isSourceAssetWormholeWrapped = useSelector(
    selectTransferIsSourceAssetWormholeWrapped
  );
  const originChain = useSelector(selectTransferOriginChain);
  const originAsset = useSelector(selectTransferOriginAsset);
  const targetChain = useSelector(selectTransferTargetChain);
  const { provider } = useEthereumProvider();
  useEffect(() => {
    if (isSourceAssetWormholeWrapped && originChain === targetChain) {
      dispatch(setTargetAsset(hexToNativeString(originAsset, originChain)));
      return;
    }
    // TODO: loading state, error state
    dispatch(setTargetAsset(undefined));
    let cancelled = false;
    (async () => {
      if (
        targetChain === CHAIN_ID_ETH &&
        provider &&
        originChain &&
        originAsset
      ) {
        const asset = await getForeignAssetEth(
          provider,
          originChain,
          originAsset
        );
        if (!cancelled) {
          dispatch(setTargetAsset(asset));
        }
      }
      if (targetChain === CHAIN_ID_SOLANA && originChain && originAsset) {
        try {
          const asset = await getForeignAssetSol(originChain, originAsset);
          if (!cancelled) {
            dispatch(setTargetAsset(asset));
          }
        } catch (e) {
          if (!cancelled) {
            // TODO: warning for this
          }
        }
      }
      if (targetChain === CHAIN_ID_TERRA && originChain && originAsset) {
        try {
          const asset = await getForeignAssetTerra(originChain, originAsset);
          if (!cancelled) {
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
    provider,
  ]);
}

export default useFetchTargetAsset;
