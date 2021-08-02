import { Connection, PublicKey } from "@solana/web3.js";
import { useEffect, useState } from "react";
import { SOLANA_HOST } from "../utils/consts";

function useSolanaBalance(
  tokenAddress: string | undefined,
  ownerAddress: PublicKey | null | undefined,
  shouldCalculate?: boolean
) {
  //TODO: should connection happen in a context?
  const [balance, setBalance] = useState<string>("");
  useEffect(() => {
    if (!tokenAddress || !ownerAddress || !shouldCalculate) {
      setBalance("");
      return;
    }
    let cancelled = false;
    const connection = new Connection(SOLANA_HOST);
    connection
      .getParsedTokenAccountsByOwner(ownerAddress, {
        mint: new PublicKey(tokenAddress),
      })
      .then(({ value }) => {
        if (!cancelled) {
          if (value.length) {
            console.log(value[0].account.data.parsed);
            setBalance(
              value[0].account.data.parsed?.info?.tokenAmount?.uiAmountString
            );
          } else {
            setBalance("0");
          }
        }
      })
      .catch(() => {
        if (!cancelled) {
          setBalance("");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [tokenAddress, ownerAddress, shouldCalculate]);
  return balance;
}

export default useSolanaBalance;
