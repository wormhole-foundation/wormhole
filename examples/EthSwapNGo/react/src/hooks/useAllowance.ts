import {
  approveEth,
  ChainId,
  getAllowanceEth,
  isEVMChain,
} from "@certusone/wormhole-sdk";
import { BigNumber } from "ethers";
import { useEffect, useMemo, useState } from "react";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";

export default function useAllowance(
  chainId: ChainId,
  contractAddress?: string,
  tokenAddress?: string,
  transferAmount?: BigInt,
  sourceIsNative?: boolean
) {
  const [allowance, setAllowance] = useState<BigInt | null>(null);
  const [isAllowanceFetching, setIsAllowanceFetching] = useState(false);
  const [isApproveProcessing, setIsApproveProcessing] = useState(false);
  const { signer } = useEthereumProvider();
  const sufficientAllowance =
    !isEVMChain(chainId) ||
    sourceIsNative ||
    (allowance && transferAmount && allowance >= transferAmount);

  useEffect(() => {
    let cancelled = false;
    if (
      isEVMChain(chainId) &&
      tokenAddress &&
      signer &&
      !isApproveProcessing &&
      contractAddress
    ) {
      setIsAllowanceFetching(true);
      getAllowanceEth(contractAddress, tokenAddress, signer).then(
        (result) => {
          if (!cancelled) {
            setIsAllowanceFetching(false);
            setAllowance(result.toBigInt());
          }
        },
        (error) => {
          if (!cancelled) {
            console.error(error);
            setIsAllowanceFetching(false);
            //setError("Unable to retrieve allowance"); //TODO set an error
          }
        }
      );
    }

    return () => {
      cancelled = true;
    };
  }, [chainId, tokenAddress, signer, isApproveProcessing, contractAddress]);

  const approveAmount: (amount: BigInt) => Promise<any> = useMemo(() => {
    return !isEVMChain(chainId) || !tokenAddress || !signer || !contractAddress
      ? (amount: BigInt) => {
          console.log(isEVMChain(chainId), tokenAddress, signer);
          console.log("hit escape in approve amount");
          return Promise.resolve();
        }
      : (amount: BigInt) => {
          setIsApproveProcessing(true);
          return approveEth(
            contractAddress,
            tokenAddress,
            signer,
            BigNumber.from(amount)
          ).then(
            () => {
              setIsApproveProcessing(false);
              return Promise.resolve();
            },
            (error) => {
              console.log(error);
              setIsApproveProcessing(false);
              return Promise.reject();
            }
          );
        };
  }, [chainId, tokenAddress, signer, contractAddress]);

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
