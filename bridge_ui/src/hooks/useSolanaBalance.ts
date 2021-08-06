import { Connection, PublicKey } from "@solana/web3.js";
import { useEffect, useState } from "react";
import { SOLANA_HOST } from "../utils/consts";

export interface Balance {
  tokenAccount: PublicKey | undefined;
  amount: string;
  decimals: number;
  uiAmount: number;
  uiAmountString: string;
}

function createBalance(
  tokenAccount: PublicKey | undefined,
  amount: string,
  decimals: number,
  uiAmount: number,
  uiAmountString: string
) {
  return {
    tokenAccount,
    amount,
    decimals,
    uiAmount,
    uiAmountString,
  };
}

function useSolanaBalance(
  tokenAddress: string | undefined,
  ownerAddress: PublicKey | null | undefined,
  shouldCalculate?: boolean
) {
  //TODO: should connection happen in a context?
  const [balance, setBalance] = useState<Balance>(
    createBalance(undefined, "", 0, 0, "")
  );
  useEffect(() => {
    if (!tokenAddress || !ownerAddress || !shouldCalculate) {
      setBalance(createBalance(undefined, "", 0, 0, ""));
      return;
    }
    let mint;
    try {
      mint = new PublicKey(tokenAddress);
    } catch (e) {
      setBalance(createBalance(undefined, "", 0, 0, ""));
      return;
    }
    let cancelled = false;
    const connection = new Connection(SOLANA_HOST, "finalized");
    connection
      .getParsedTokenAccountsByOwner(ownerAddress, { mint })
      .then(({ value }) => {
        if (!cancelled) {
          if (value.length) {
            setBalance(
              createBalance(
                value[0].pubkey,
                value[0].account.data.parsed?.info?.tokenAmount?.amount,
                value[0].account.data.parsed?.info?.tokenAmount?.decimals,
                value[0].account.data.parsed?.info?.tokenAmount?.uiAmount,
                value[0].account.data.parsed?.info?.tokenAmount?.uiAmountString
              )
            );
          } else {
            setBalance(createBalance(undefined, "0", 0, 0, "0"));
          }
        }
      })
      .catch(() => {
        if (!cancelled) {
          setBalance(createBalance(undefined, "", 0, 0, ""));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [tokenAddress, ownerAddress, shouldCalculate]);
  return balance;
}

export default useSolanaBalance;
