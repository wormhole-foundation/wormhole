import axios from "axios";
import { useEffect, useMemo, useState } from "react";
import { DataWrapper } from "../store/helpers";
import {
  RECENT_TRANSACTIONS_WORMHOLE,
  TOTAL_TRANSACTIONS_WORMHOLE,
  VAA_EMITTER_ADDRESSES,
} from "../utils/consts";

export type TransactionCount = {
  totalAllTime: number;
  total48h: number;
  mostRecent: any; //This will be a signedVAA
};

const mergeResults = (totals: any, recents: any): TransactionCount | null => {
  let totalAllTime = 0;
  let total48h = 0;
  const lastDays = Object.values(totals?.DailyTotals || {});
  const lastTwoDays: any = lastDays.slice(lastDays.length - 2);
  VAA_EMITTER_ADDRESSES.forEach((address: string) => {
    let totalAll = (totals?.TotalCount && totals.TotalCount[address]) || 0;
    let total48 =
      lastTwoDays.length === 2
        ? (lastTwoDays[0][address] || 0) + (lastTwoDays[1][address] || 0)
        : 0;

    totalAllTime += totalAll;
    total48h += total48;
  });

  return {
    totalAllTime,
    total48h,
    mostRecent: null,
  };
};

const useTransactionCount = (): DataWrapper<TransactionCount> => {
  const [totals, setTotals] = useState(null);
  const [recents, setRecents] = useState(null);

  const [loadingTotals, setLoadingTotals] = useState(false);
  const [loadingRecents, setLoadingRecents] = useState(false);

  const [totalsError, setTotalsError] = useState("");
  const [recentsError, setRecentsError] = useState("");

  useEffect(() => {
    let cancelled = false;
    setLoadingTotals(true);
    axios.get(TOTAL_TRANSACTIONS_WORMHOLE).then(
      (results) => {
        if (!cancelled) {
          setTotals(results.data);
          setLoadingTotals(false);
        }
      },
      (error) => {
        if (!cancelled) {
          setTotalsError("Unable to retrieve transaction totals.");
          setLoadingTotals(false);
        }
      }
    );
  }, []);

  useEffect(() => {
    let cancelled = false;
    setLoadingRecents(true);
    axios.get(RECENT_TRANSACTIONS_WORMHOLE).then(
      (results) => {
        if (!cancelled) {
          setRecents(results.data);
          setLoadingRecents(false);
        }
      },
      (error) => {
        if (!cancelled) {
          setRecentsError("Unable to retrieve recent transactions.");
          setLoadingRecents(false);
        }
      }
    );
  }, []);

  return useMemo(() => {
    const data = mergeResults(totals, recents);
    return {
      isFetching: loadingRecents || loadingTotals,
      error: totalsError || recentsError,
      receivedAt: null,
      data: data,
    };
  }, [
    totals,
    recents,
    loadingRecents,
    loadingTotals,
    recentsError,
    totalsError,
  ]);
};

export default useTransactionCount;
