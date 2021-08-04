import { ethers } from "ethers";
import { formatUnits } from "ethers/lib/utils";
import { useEffect, useState } from "react";
import { TokenImplementation__factory } from "../ethers-contracts";

function useEthereumBalance(
  address: string | undefined,
  provider: ethers.providers.Web3Provider | undefined,
  shouldCalculate?: boolean
) {
  //TODO: should this check allowance too or subtract allowance?
  const [balance, setBalance] = useState<string>("");
  useEffect(() => {
    if (!address || !provider || !shouldCalculate) {
      setBalance("");
      return;
    }
    let cancelled = false;
    const token = TokenImplementation__factory.connect(address, provider);
    token
      .decimals()
      .then((decimals) => {
        provider
          ?.getSigner()
          .getAddress()
          .then((pk) => {
            token.balanceOf(pk).then((n) => {
              if (!cancelled) {
                setBalance(formatUnits(n, decimals));
              }
            });
          });
      })
      .catch(() => {
        if (!cancelled) {
          setBalance("");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [address, provider, shouldCalculate]);
  return balance;
}

export default useEthereumBalance;
