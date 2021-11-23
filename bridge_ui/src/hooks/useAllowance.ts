import {
  approveEth,
  ChainId,
  getAllowanceEth,
  isEVMChain,
} from "@certusone/wormhole-sdk";
import { BigNumber } from "ethers";
import { useEffect, useMemo, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { selectTransferIsApproving } from "../store/selectors";
import { setIsApproving } from "../store/transferSlice";
import { getTokenBridgeAddressForChain } from "../utils/consts";

export default function useAllowance(
  chainId: ChainId,
  tokenAddress?: string,
  transferAmount?: BigInt,
  sourceIsNative?: boolean
) {
  const dispatch = useDispatch();
  const [allowance, setAllowance] = useState<BigInt | null>(null);
  const [isAllowanceFetching, setIsAllowanceFetching] = useState(false);
  const isApproveProcessing = useSelector(selectTransferIsApproving);
  const { signer } = useEthereumProvider();
  const sufficientAllowance =
    !isEVMChain(chainId) ||
    sourceIsNative ||
    (allowance && transferAmount && allowance >= transferAmount);

  useEffect(() => {
    let cancelled = false;
    if (isEVMChain(chainId) && tokenAddress && signer && !isApproveProcessing) {
      setIsAllowanceFetching(true);
      getAllowanceEth(
        getTokenBridgeAddressForChain(chainId),
        tokenAddress,
        signer
      ).then(
        (result) => {
          if (!cancelled) {
            setIsAllowanceFetching(false);
            setAllowance(result.toBigInt());
          }
        },
        (error) => {
          if (!cancelled) {
            setIsAllowanceFetching(false);
            //setError("Unable to retrieve allowance"); //TODO set an error
          }
        }
      );
    }

    return () => {
      cancelled = true;
    };
  }, [chainId, tokenAddress, signer, isApproveProcessing]);

  const approveAmount: (amount: BigInt) => Promise<any> = useMemo(() => {
    return !isEVMChain(chainId) || !tokenAddress || !signer
      ? (amount: BigInt) => {
          return Promise.resolve();
        }
      : (amount: BigInt) => {
          dispatch(setIsApproving(true));
          return approveEth(
            getTokenBridgeAddressForChain(chainId),
            tokenAddress,
            signer,
            BigNumber.from(amount)
          ).then(
            () => {
              dispatch(setIsApproving(false));
              return Promise.resolve();
            },
            () => {
              dispatch(setIsApproving(false));
              return Promise.reject();
            }
          );
        };
  }, [chainId, tokenAddress, signer, dispatch]);

  return useMemo(
    () => ({
      sufficientAllowance,
      approveAmount,
      isAllowanceFetching,
      isApproveProcessing,
    }),
    [
      sufficientAllowance,
      approveAmount,
      isAllowanceFetching,
      isApproveProcessing,
    ]
  );
}
