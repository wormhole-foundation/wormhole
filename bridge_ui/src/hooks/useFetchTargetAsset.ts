import {
  ChainId,
  CHAIN_ID_ALGORAND,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA2,
  getForeignAssetAlgorand,
  getForeignAssetEth,
  getForeignAssetSolana,
  getForeignAssetTerra,
  hexToNativeAssetString,
  hexToUint8Array,
  isEVMChain,
  isTerraChain,
} from "@certusone/wormhole-sdk";
import {
  getForeignAssetEth as getForeignAssetEthNFT,
  getForeignAssetSol as getForeignAssetSolNFT,
} from "@certusone/wormhole-sdk/lib/esm/nft_bridge";
import { BigNumber } from "@ethersproject/bignumber";
import { arrayify } from "@ethersproject/bytes";
import { Connection } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import algosdk from "algosdk";
import { ethers } from "ethers";
import { useCallback, useEffect, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import {
  errorDataWrapper,
  fetchDataWrapper,
  receiveDataWrapper,
} from "../store/helpers";
import { setTargetAsset as setNFTTargetAsset } from "../store/nftSlice";
import {
  selectNFTIsSourceAssetWormholeWrapped,
  selectNFTOriginAsset,
  selectNFTOriginChain,
  selectNFTOriginTokenId,
  selectNFTTargetChain,
  selectTransferIsSourceAssetWormholeWrapped,
  selectTransferOriginAsset,
  selectTransferOriginChain,
  selectTransferTargetChain,
} from "../store/selectors";
import { setTargetAsset as setTransferTargetAsset } from "../store/transferSlice";
import {
  ALGORAND_HOST,
  ALGORAND_TOKEN_BRIDGE_ID,
  getEvmChainId,
  getNFTBridgeAddressForChain,
  getTokenBridgeAddressForChain,
  SOLANA_HOST,
  SOL_NFT_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  getTerraConfig,
} from "../utils/consts";
import { queryExternalId } from "../utils/terra";

function useFetchTargetAsset(nft?: boolean) {
  const dispatch = useDispatch();
  const isSourceAssetWormholeWrapped = useSelector(
    nft
      ? selectNFTIsSourceAssetWormholeWrapped
      : selectTransferIsSourceAssetWormholeWrapped
  );
  const originChain = useSelector(
    nft ? selectNFTOriginChain : selectTransferOriginChain
  );
  const originAsset = useSelector(
    nft ? selectNFTOriginAsset : selectTransferOriginAsset
  );
  const originTokenId = useSelector(selectNFTOriginTokenId);
  const tokenId = originTokenId || ""; // this should exist by this step for NFT transfers
  const targetChain = useSelector(
    nft ? selectNFTTargetChain : selectTransferTargetChain
  );
  const setTargetAsset = nft ? setNFTTargetAsset : setTransferTargetAsset;
  const { provider, chainId: evmChainId } = useEthereumProvider();
  const correctEvmNetwork = getEvmChainId(targetChain);
  const hasCorrectEvmNetwork = evmChainId === correctEvmNetwork;
  const [lastSuccessfulArgs, setLastSuccessfulArgs] = useState<{
    isSourceAssetWormholeWrapped: boolean | undefined;
    originChain: ChainId | undefined;
    originAsset: string | undefined;
    targetChain: ChainId;
    nft?: boolean;
    tokenId?: string;
  } | null>(null);
  const argsMatchLastSuccess =
    !!lastSuccessfulArgs &&
    lastSuccessfulArgs.isSourceAssetWormholeWrapped ===
      isSourceAssetWormholeWrapped &&
    lastSuccessfulArgs.originChain === originChain &&
    lastSuccessfulArgs.originAsset === originAsset &&
    lastSuccessfulArgs.targetChain === targetChain &&
    lastSuccessfulArgs.nft === nft &&
    lastSuccessfulArgs.tokenId === tokenId;
  const setArgs = useCallback(
    () =>
      setLastSuccessfulArgs({
        isSourceAssetWormholeWrapped,
        originChain,
        originAsset,
        targetChain,
        nft,
        tokenId,
      }),
    [
      isSourceAssetWormholeWrapped,
      originChain,
      originAsset,
      targetChain,
      nft,
      tokenId,
    ]
  );
  useEffect(() => {
    if (argsMatchLastSuccess) {
      return;
    }
    setLastSuccessfulArgs(null);
    let cancelled = false;
    (async () => {
      if (isSourceAssetWormholeWrapped && originChain === targetChain) {
        if (originChain === CHAIN_ID_TERRA2) {
          const tokenId = await queryExternalId(originAsset || "");
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                receiveDataWrapper({
                  doesExist: true,
                  address: tokenId || null,
                })
              )
            );
          }
        } else {
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                receiveDataWrapper({
                  doesExist: true,
                  address:
                    hexToNativeAssetString(originAsset, originChain) || null,
                })
              )
            );
          }
        }
        if (!cancelled) {
          setArgs();
        }
        return;
      }
      if (
        isEVMChain(targetChain) &&
        provider &&
        hasCorrectEvmNetwork &&
        originChain &&
        originAsset
      ) {
        dispatch(setTargetAsset(fetchDataWrapper()));
        try {
          const asset = await (nft
            ? getForeignAssetEthNFT(
                getNFTBridgeAddressForChain(targetChain),
                provider,
                originChain,
                hexToUint8Array(originAsset)
              )
            : getForeignAssetEth(
                getTokenBridgeAddressForChain(targetChain),
                provider,
                originChain,
                hexToUint8Array(originAsset)
              ));
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                receiveDataWrapper({
                  doesExist: asset !== ethers.constants.AddressZero,
                  address: asset,
                })
              )
            );
            setArgs();
          }
        } catch (e) {
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                errorDataWrapper(
                  "Unable to determine existence of wrapped asset"
                )
              )
            );
          }
        }
      }
      if (targetChain === CHAIN_ID_SOLANA && originChain && originAsset) {
        dispatch(setTargetAsset(fetchDataWrapper()));
        try {
          const connection = new Connection(SOLANA_HOST, "confirmed");
          const asset = await (nft
            ? getForeignAssetSolNFT(
                SOL_NFT_BRIDGE_ADDRESS,
                originChain,
                hexToUint8Array(originAsset),
                arrayify(BigNumber.from(tokenId || "0"))
              )
            : getForeignAssetSolana(
                connection,
                SOL_TOKEN_BRIDGE_ADDRESS,
                originChain,
                hexToUint8Array(originAsset)
              ));
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                receiveDataWrapper({ doesExist: !!asset, address: asset })
              )
            );
            setArgs();
          }
        } catch (e) {
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                errorDataWrapper(
                  "Unable to determine existence of wrapped asset"
                )
              )
            );
          }
        }
      }
      if (isTerraChain(targetChain) && originChain && originAsset) {
        dispatch(setTargetAsset(fetchDataWrapper()));
        try {
          const lcd = new LCDClient(getTerraConfig(targetChain));
          const asset = await getForeignAssetTerra(
            getTokenBridgeAddressForChain(targetChain),
            lcd,
            originChain,
            hexToUint8Array(originAsset)
          );
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                receiveDataWrapper({ doesExist: !!asset, address: asset })
              )
            );
            setArgs();
          }
        } catch (e) {
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                errorDataWrapper(
                  "Unable to determine existence of wrapped asset"
                )
              )
            );
          }
        }
      }
      if (targetChain === CHAIN_ID_ALGORAND && originChain && originAsset) {
        dispatch(setTargetAsset(fetchDataWrapper()));
        try {
          const algodClient = new algosdk.Algodv2(
            ALGORAND_HOST.algodToken,
            ALGORAND_HOST.algodServer,
            ALGORAND_HOST.algodPort
          );
          const asset = await getForeignAssetAlgorand(
            algodClient,
            ALGORAND_TOKEN_BRIDGE_ID,
            originChain,
            originAsset
          );
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                receiveDataWrapper({
                  doesExist: !!asset,
                  address: asset === null ? asset : asset.toString(),
                })
              )
            );
            setArgs();
          }
        } catch (e) {
          console.error(e);
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                errorDataWrapper(
                  "Unable to determine existence of wrapped asset"
                )
              )
            );
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
    nft,
    setTargetAsset,
    tokenId,
    hasCorrectEvmNetwork,
    argsMatchLastSuccess,
    setArgs,
  ]);
}

export default useFetchTargetAsset;
