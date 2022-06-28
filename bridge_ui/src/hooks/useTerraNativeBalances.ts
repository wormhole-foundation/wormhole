import { TerraChainId } from "@certusone/wormhole-sdk";
import { LCDClient } from "@terra-money/terra.js";
import { MutableRefObject, useEffect, useMemo, useState } from "react";
import { getTerraConfig } from "../utils/consts";

export interface TerraNativeBalances {
  [index: string]: string;
}

export default function useTerraNativeBalances(
  chainId: TerraChainId,
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
      const lcd = new LCDClient(getTerraConfig(chainId));
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
  }, [walletAddress, refresh, chainId]);
  const value = useMemo(() => ({ isLoading, balances }), [isLoading, balances]);
  return value;
}
