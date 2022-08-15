import axios from "axios";
import { useEffect, useMemo, useRef, useState } from "react";
import { useSelector } from "react-redux";
import {
  selectTransferAmount,
  selectTransferOriginAsset,
  selectTransferOriginChain,
  selectTransferSourceChain,
} from "../store/selectors";
import { WORMHOLE_RPC_HOSTS } from "../utils/consts";
import { ChainId } from "@certusone/wormhole-sdk";

const REMAINING_NOTIONAL_TOLERANCE = 0.95;
interface TokenListEntry {
  originAddress: string;
  originChainId: number;
  price: number;
}

interface TokenList {
  entries: TokenListEntry[];
}

interface AvailableNotionalByChainEntry {
  chainId: number;
  remainingAvailableNotional: number;
  notionalLimit: number;
}

interface AvailableNotionalByChain {
  entries: AvailableNotionalByChainEntry[];
}

export interface ChainLimits {
  chainId: ChainId;
  chainNotionalLimit: number;
  chainRemainingAvailableNotional: number;
  tokenPrice: number;
}

export interface IsTransferLimitedResult {
  isLimited: boolean;
  reason?: "EXCEEDS_REMAINING_NOTIONAL" | "EXCEEDS_MAX_NOTIONAL";
  limits?: ChainLimits;
}

const useIsTransferLimited = (): IsTransferLimitedResult => {
  const [tokenList, setTokenList] = useState<TokenList | null>(null);
  const [availableNotionalByChain, setAvailableNotionalByChain] =
    useState<AvailableNotionalByChain | null>(null);

  const sourceChain = useSelector(selectTransferSourceChain);
  const originChain = useSelector(selectTransferOriginChain);
  const originAsset = useSelector(selectTransferOriginAsset);
  const amount = useSelector(selectTransferAmount);
  const amountParsed = useMemo(() => {
    return amount ? parseFloat(amount) : undefined;
  }, [amount]);

  const effectTriggered = useRef(false);

  useEffect(() => {
    if (!effectTriggered.current && amountParsed) {
      let cancelled = false;
      (async () => {
        for (const rpcHost of WORMHOLE_RPC_HOSTS) {
          try {
            const baseUrl = `${rpcHost}/v1/governor`;
            const [tokenListResponse, availableNotionalByChainResponse] =
              await Promise.all([
                axios.get<TokenList>(`${baseUrl}/token_list`),
                axios.get<AvailableNotionalByChain>(
                  `${baseUrl}/available_notional_by_chain`
                ),
              ]);
            if (!cancelled) {
              setTokenList(tokenListResponse.data);
              setAvailableNotionalByChain(
                availableNotionalByChainResponse.data
              );
              break;
            }
          } catch (error) {
            console.error(error);
          }
          if (cancelled) {
            break;
          }
        }
        return () => {
          cancelled = true;
        };
      })();
      effectTriggered.current = true;
    }
  }, [amountParsed]);

  const result = useMemo<IsTransferLimitedResult>(() => {
    if (
      originAsset &&
      originChain &&
      amountParsed &&
      tokenList &&
      availableNotionalByChain
    ) {
      const token = tokenList.entries.find(
        (entry) =>
          entry.originChainId === originChain &&
          entry.originAddress === "0x" + originAsset
      );
      if (token) {
        const chain = availableNotionalByChain.entries.find(
          (entry) => entry.chainId === sourceChain
        );
        if (chain) {
          const transferNotional = token.price * amountParsed;
          const isLimitedReason =
            transferNotional > chain.notionalLimit
              ? "EXCEEDS_MAX_NOTIONAL"
              : transferNotional >
                chain.remainingAvailableNotional * REMAINING_NOTIONAL_TOLERANCE
              ? "EXCEEDS_REMAINING_NOTIONAL"
              : undefined;
          return {
            isLimited: !!isLimitedReason,
            reason: isLimitedReason,
            limits: {
              chainId: sourceChain,
              chainNotionalLimit: chain.notionalLimit,
              chainRemainingAvailableNotional: chain.remainingAvailableNotional,
              tokenPrice: token.price,
            },
          };
        }
      }
    }
    return {
      isLimited: false,
    };
  }, [
    sourceChain,
    originChain,
    originAsset,
    amountParsed,
    tokenList,
    availableNotionalByChain,
  ]);

  return result;
};

export default useIsTransferLimited;
