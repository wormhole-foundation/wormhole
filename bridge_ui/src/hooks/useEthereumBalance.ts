import { ethers } from "ethers";
import { formatUnits } from "ethers/lib/utils";
import { useEffect, useState } from "react";
import { TokenImplementation__factory } from "../ethers-contracts";

// TODO: can this be shared with other balances
export interface Balance {
  decimals: number;
  uiAmountString: string;
}

function createBalance(decimals: number, uiAmountString: string) {
  return {
    decimals,
    uiAmountString,
  };
}

function useEthereumBalance(
  address: string | undefined,
  ownerAddress: string | undefined,
  provider: ethers.providers.Web3Provider | undefined,
  shouldCalculate?: boolean
) {
  //TODO: should this check allowance too or subtract allowance?
  const [balance, setBalance] = useState<Balance>(createBalance(0, ""));
  useEffect(() => {
    if (!address || !ownerAddress || !provider || !shouldCalculate) {
      setBalance(createBalance(0, ""));
      return;
    }
    let cancelled = false;
    const token = TokenImplementation__factory.connect(address, provider);
    token
      .decimals()
      .then((decimals) => {
        token.balanceOf(ownerAddress).then((n) => {
          if (!cancelled) {
            setBalance(createBalance(decimals, formatUnits(n, decimals)));
          }
        });
      })
      .catch(() => {
        if (!cancelled) {
          setBalance(createBalance(0, ""));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [address, ownerAddress, provider, shouldCalculate]);
  return balance;
}

export default useEthereumBalance;
