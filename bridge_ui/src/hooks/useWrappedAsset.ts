import { ethers } from "ethers";
import { useEffect, useState } from "react";
import { ChainId, CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "../utils/consts";
import {
  getAttestedAssetEth,
  getAttestedAssetSol,
} from "../utils/getAttestedAsset";
export interface WrappedAssetState {
  isLoading: boolean;
  isWrapped: boolean;
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
    isWrapped: false,
    wrappedAsset: null,
  });
  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (checkChain === CHAIN_ID_ETH && provider) {
        setState({ isLoading: true, isWrapped: false, wrappedAsset: null });
        const asset = await getAttestedAssetEth(
          provider,
          originChain,
          originAsset
        );
        if (!cancelled) {
          setState({
            isLoading: false,
            isWrapped: !!asset && asset !== ethers.constants.AddressZero,
            wrappedAsset: asset,
          });
        }
      } else if (checkChain === CHAIN_ID_SOLANA) {
        setState({ isLoading: true, isWrapped: false, wrappedAsset: null });
        try {
          const asset = await getAttestedAssetSol(originChain, originAsset);
          if (!cancelled) {
            setState({
              isLoading: false,
              isWrapped: !!asset,
              wrappedAsset: asset,
            });
          }
        } catch (e) {
          if (!cancelled) {
            // TODO: warning for this
            setState({
              isLoading: false,
              isWrapped: false,
              wrappedAsset: null,
            });
          }
        }
      } else {
        setState({ isLoading: false, isWrapped: false, wrappedAsset: null });
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [checkChain, originChain, originAsset, provider]);
  return state;
}

export default useWrappedAsset;
