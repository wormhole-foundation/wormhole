import { LCDClient } from "@terra-money/terra.js";
import { MutableRefObject, useEffect, useMemo, useState } from "react";
import { TERRA_HOST } from "../utils/consts";

export interface TerraNativeBalances {
  [index: string]: string;
}

export default function useTerraNativeBalances(
  walletAddress?: string,
  refreshRef?: MutableRefObject<() => void>
) {
  const [isLoading, setIsLoading] = useState(true);
  const [balances, setBalances] = useState<TerraNativeBalances | undefined>({});
  const [refresh, setRefresh] = useState(false);
  useEffect(() => {
    if (refreshRef) {
      refreshRef.current = () => {
        setRefresh(true);
      };
    }
  }, [refreshRef]);
  useEffect(() => {
    setRefresh(false);
    if (walletAddress) {
      setIsLoading(true);
      setBalances(undefined);
      const lcd = new LCDClient(TERRA_HOST);
      lcd.bank
        .balance(walletAddress)
        .then(([coins]) => {
          // coins doesn't support reduce
          const balancePairs = coins.map(({ amount, denom }) => [
            denom,
            amount,
          ]);
          const balance = balancePairs.reduce((obj, current) => {
            obj[current[0].toString()] = current[1].toString();
            return obj;
          }, {} as TerraNativeBalances);
          setIsLoading(false);
          setBalances(balance);
        })
        .catch((e) => {
          setIsLoading(false);
          setBalances(undefined);
        });
    } else {
      setIsLoading(false);
      setBalances(undefined);
    }
  }, [walletAddress, refresh]);
  const value = useMemo(() => ({ isLoading, balances }), [isLoading, balances]);
  return value;
}
