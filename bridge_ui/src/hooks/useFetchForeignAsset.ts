import {
  ChainId,
  CHAIN_ID_TERRA,
  getForeignAssetEth,
  getForeignAssetSolana,
  getForeignAssetTerra,
  nativeToHexString,
  hexToUint8Array,
} from "@certusone/wormhole-sdk";
import { Connection } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { ethers } from "ethers";
import { useEffect, useMemo, useState } from "react";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { DataWrapper } from "../store/helpers";
import {
  getEvmChainId,
  getTokenBridgeAddressForChain,
  SOLANA_HOST,
  SOL_TOKEN_BRIDGE_ADDRESS,
  TERRA_HOST,
  TERRA_TOKEN_BRIDGE_ADDRESS,
} from "../utils/consts";
import { isEVMChain } from "../utils/ethereum";
import useIsWalletReady from "./useIsWalletReady";

export type ForeignAssetInfo = {
  doesExist: boolean;
  address: string | null;
};

function useFetchForeignAsset(
  originChain: ChainId,
  originAsset: string,
  foreignChain: ChainId
): DataWrapper<ForeignAssetInfo> {
  const { provider, chainId: evmChainId } = useEthereumProvider();
  const { isReady, statusMessage } = useIsWalletReady(foreignChain);
  const correctEvmNetwork = getEvmChainId(foreignChain);
  const hasCorrectEvmNetwork = evmChainId === correctEvmNetwork;

  const [assetAddress, setAssetAddress] = useState<string | null>(null);
  const [doesExist, setDoesExist] = useState(false);
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const originAssetHex = useMemo(
    () => nativeToHexString(originAsset, originChain),
    [originAsset, originChain]
  );

  const argumentError = useMemo(
    () =>
      !foreignChain ||
      !originAssetHex ||
      foreignChain === originChain ||
      (isEVMChain(foreignChain) && !isReady) ||
      (isEVMChain(foreignChain) && !hasCorrectEvmNetwork),
    [isReady, foreignChain, originChain, hasCorrectEvmNetwork, originAssetHex]
  );

  useEffect(() => {
    if (argumentError || !originAssetHex) {
      return;
    }

    let cancelled = false;
    setIsLoading(true);
    setAssetAddress(null);
    setError("");
    setDoesExist(false);
    const getterFunc: () => Promise<string | null> = isEVMChain(foreignChain)
      ? () =>
          getForeignAssetEth(
            getTokenBridgeAddressForChain(foreignChain),
            provider as any, //why does this typecheck work elsewhere?
            originChain,
            hexToUint8Array(originAssetHex)
          )
      : foreignChain === CHAIN_ID_TERRA
      ? () => {
          const lcd = new LCDClient(TERRA_HOST);
          return getForeignAssetTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            lcd,
            originChain,
            hexToUint8Array(originAssetHex)
          );
        }
      : () => {
          const connection = new Connection(SOLANA_HOST, "confirmed");
          return getForeignAssetSolana(
            connection,
            SOL_TOKEN_BRIDGE_ADDRESS,
            originChain,
            hexToUint8Array(originAssetHex)
          );
        };

    const promise = getterFunc();

    promise
      .then((result) => {
        if (!cancelled) {
          if (
            result &&
            !(
              isEVMChain(foreignChain) &&
              result === ethers.constants.AddressZero
            )
          ) {
            setDoesExist(true);
            setIsLoading(false);
            setAssetAddress(result);
          } else {
            setDoesExist(false);
            setIsLoading(false);
            setAssetAddress(null);
          }
        }
      })
      .catch((e) => {
        if (!cancelled) {
          setError("Could not retrieve the foreign asset.");
          setIsLoading(false);
        }
      });
  }, [argumentError, foreignChain, originAssetHex, originChain, provider]);

  const compoundError = useMemo(() => {
    return error
      ? error
      : !isReady
      ? statusMessage
      : argumentError
      ? "Invalid arguments."
      : "";
  }, [error, isReady, statusMessage, argumentError]);

  const output: DataWrapper<ForeignAssetInfo> = useMemo(
    () => ({
      error: compoundError,
      isFetching: isLoading,
      data: { address: assetAddress, doesExist },
      receivedAt: null,
    }),
    [compoundError, isLoading, assetAddress, doesExist]
  );

  return output;
}

export default useFetchForeignAsset;
