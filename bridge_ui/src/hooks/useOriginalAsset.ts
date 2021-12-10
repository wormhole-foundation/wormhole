import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  getOriginalAssetEth,
  getOriginalAssetSol,
  getOriginalAssetTerra,
  hexToNativeString,
  isEVMChain,
  uint8ArrayToHex,
  uint8ArrayToNative,
} from "@certusone/wormhole-sdk";
import {
  getOriginalAssetEth as getOriginalAssetEthNFT,
  getOriginalAssetSol as getOriginalAssetSolNFT,
  WormholeWrappedNFTInfo,
} from "@certusone/wormhole-sdk/lib/esm/nft_bridge";
import { Web3Provider } from "@ethersproject/providers";
import { ethers } from "ethers";
import { Connection } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Provider,
  useEthereumProvider,
} from "../contexts/EthereumProviderContext";
import { DataWrapper } from "../store/helpers";
import {
  getNFTBridgeAddressForChain,
  getTokenBridgeAddressForChain,
  SOLANA_HOST,
  SOLANA_SYSTEM_PROGRAM_ADDRESS,
  SOL_NFT_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  TERRA_HOST,
} from "../utils/consts";
import useIsWalletReady from "./useIsWalletReady";

export type OriginalAssetInfo = {
  originChain: ChainId | null;
  originAddress: string | null;
  originTokenId: string | null;
};

export async function getOriginalAssetToken(
  foreignChain: ChainId,
  foreignNativeStringAddress: string,
  provider?: Web3Provider
) {
  let promise = null;
  try {
    if (isEVMChain(foreignChain) && provider) {
      promise = await getOriginalAssetEth(
        getTokenBridgeAddressForChain(foreignChain),
        provider,
        foreignNativeStringAddress,
        foreignChain
      );
    } else if (foreignChain === CHAIN_ID_SOLANA) {
      const connection = new Connection(SOLANA_HOST, "confirmed");
      promise = await getOriginalAssetSol(
        connection,
        SOL_TOKEN_BRIDGE_ADDRESS,
        foreignNativeStringAddress
      );
    } else if (foreignChain === CHAIN_ID_TERRA) {
      const lcd = new LCDClient(TERRA_HOST);
      promise = await getOriginalAssetTerra(lcd, foreignNativeStringAddress);
    }
  } catch (e) {
    promise = Promise.reject("Invalid foreign arguments.");
  }
  if (!promise) {
    promise = Promise.reject("Invalid foreign arguments.");
  }
  return promise;
}

export async function getOriginalAssetNFT(
  foreignChain: ChainId,
  foreignNativeStringAddress: string,
  tokenId?: string,
  provider?: Provider
) {
  let promise = null;
  try {
    if (isEVMChain(foreignChain) && provider && tokenId) {
      promise = getOriginalAssetEthNFT(
        getNFTBridgeAddressForChain(foreignChain),
        provider,
        foreignNativeStringAddress,
        tokenId,
        foreignChain
      );
    } else if (foreignChain === CHAIN_ID_SOLANA) {
      const connection = new Connection(SOLANA_HOST, "confirmed");
      promise = getOriginalAssetSolNFT(
        connection,
        SOL_NFT_BRIDGE_ADDRESS,
        foreignNativeStringAddress
      );
    }
  } catch (e) {
    promise = Promise.reject("Invalid foreign arguments.");
  }
  if (!promise) {
    promise = Promise.reject("Invalid foreign arguments.");
  }
  return promise;
}

//TODO refactor useCheckIfWormholeWrapped to use this function, and probably move to SDK
export async function getOriginalAsset(
  foreignChain: ChainId,
  foreignNativeStringAddress: string,
  nft: boolean,
  tokenId?: string,
  provider?: Provider
): Promise<WormholeWrappedNFTInfo> {
  const result = nft
    ? await getOriginalAssetNFT(
        foreignChain,
        foreignNativeStringAddress,
        tokenId,
        provider
      )
    : await getOriginalAssetToken(
        foreignChain,
        foreignNativeStringAddress,
        provider
      );

  if (
    isEVMChain(result.chainId) &&
    uint8ArrayToNative(result.assetAddress, result.chainId) ===
      ethers.constants.AddressZero
  ) {
    throw new Error("Unable to find address.");
  }
  if (
    result.chainId === CHAIN_ID_SOLANA &&
    uint8ArrayToNative(result.assetAddress, result.chainId) ===
      SOLANA_SYSTEM_PROGRAM_ADDRESS
  ) {
    throw new Error("Unable to find address.");
  }

  return result;
}

//This potentially returns the same chain as the foreign chain, in the case where the asset is native
function useOriginalAsset(
  foreignChain: ChainId,
  foreignAddress: string,
  nft: boolean,
  tokenId?: string
): DataWrapper<OriginalAssetInfo> {
  const { provider } = useEthereumProvider();
  const { isReady } = useIsWalletReady(foreignChain, false);
  const [originAddress, setOriginAddress] = useState<string | null>(null);
  const [originTokenId, setOriginTokenId] = useState<string | null>(null);
  const [originChain, setOriginChain] = useState<ChainId | null>(null);
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [previousArgs, setPreviousArgs] = useState<{
    foreignChain: ChainId;
    foreignAddress: string;
    nft: boolean;
    tokenId?: string;
  } | null>(null);
  const argsEqual =
    !!previousArgs &&
    previousArgs.foreignChain === foreignChain &&
    previousArgs.foreignAddress === foreignAddress &&
    previousArgs.nft === nft &&
    previousArgs.tokenId === tokenId;
  const setArgs = useCallback(
    () => setPreviousArgs({ foreignChain, foreignAddress, nft, tokenId }),
    [foreignChain, foreignAddress, nft, tokenId]
  );

  const argumentError = useMemo(
    () =>
      !foreignChain ||
      !foreignAddress ||
      (isEVMChain(foreignChain) && !isReady) ||
      (isEVMChain(foreignChain) && nft && !tokenId) ||
      argsEqual,
    [isReady, nft, tokenId, argsEqual, foreignChain, foreignAddress]
  );

  useEffect(() => {
    if (!argsEqual) {
      setError("");
      setOriginAddress(null);
      setOriginTokenId(null);
      setOriginChain(null);
      setPreviousArgs(null);
    }
    if (argumentError) {
      return;
    }
    let cancelled = false;
    setIsLoading(true);

    getOriginalAsset(foreignChain, foreignAddress, nft, tokenId, provider)
      .then((result) => {
        if (!cancelled) {
          setIsLoading(false);
          setArgs();
          setOriginAddress(
            hexToNativeString(
              uint8ArrayToHex(result.assetAddress),
              result.chainId
            ) || null
          );
          setOriginTokenId(result.tokenId || null);
          setOriginChain(result.chainId);
        }
      })
      .catch((e) => {
        if (!cancelled) {
          setIsLoading(false);
          setError("Unable to determine original asset.");
        }
      });
  }, [
    foreignChain,
    foreignAddress,
    nft,
    provider,
    setArgs,
    argumentError,
    tokenId,
    argsEqual,
  ]);

  const output: DataWrapper<OriginalAssetInfo> = useMemo(
    () => ({
      error: error,
      isFetching: isLoading,
      data:
        originChain || originAddress || originTokenId
          ? { originChain, originAddress, originTokenId }
          : null,
      receivedAt: null,
    }),
    [isLoading, originAddress, originChain, originTokenId, error]
  );

  return output;
}

export default useOriginalAsset;
