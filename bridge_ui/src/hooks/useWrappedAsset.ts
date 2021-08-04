import { ethers } from "ethers";
import { useEffect, useState } from "react";
import { ChainId, CHAIN_ID_ETH } from "../utils/consts";
import { wrappedAssetEth } from "../utils/wrappedAsset";

export interface WrappedAssetState {
  isLoading: boolean;
  wrappedAsset: string | null;
}

function useWrappedAsset(
  checkChain: ChainId,
  originChain: ChainId,
  originAsset: string,
  provider: ethers.providers.Web3Provider | undefined
) {
  const [state, setState] = useState<WrappedAssetState>({
    isLoading: false,
    wrappedAsset: null,
  });
  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (provider && checkChain === CHAIN_ID_ETH) {
        setState({ isLoading: true, wrappedAsset: null });
        const asset = await wrappedAssetEth(provider, originChain, originAsset);
        if (!cancelled) {
          setState({ isLoading: false, wrappedAsset: asset });
        }
      } else {
        setState({ isLoading: false, wrappedAsset: null });
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [checkChain, originChain, originAsset, provider]);
  return state;
}

export default useWrappedAsset;
