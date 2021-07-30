import { ethers } from "ethers";
import { formatUnits } from "ethers/lib/utils";
import { useEffect, useState } from "react";
import { TokenImplementation__factory } from "../ethers-contracts";

function useEthereumBalance(address: string, provider?: ethers.providers.Web3Provider) {
  //TODO: should this check allowance too or subtract allowance?
  const [balance, setBalance] = useState<string>('')
  useEffect(()=>{
    if (!address || !provider) {
      setBalance('')
      return
    }
    let cancelled = false
    const token = TokenImplementation__factory.connect(address, provider);
    token.decimals().then((decimals) => {
      console.log(decimals);
      provider
      ?.getSigner()
      .getAddress()
      .then((pk) => {
        console.log(pk)
        token.balanceOf(pk).then((n) => {
          if (!cancelled) {
            setBalance(formatUnits(n,decimals))
          }
        });
      });
    });
    return () => {
      cancelled = true
    }
  },[address, provider])
  return balance
}

export default useEthereumBalance