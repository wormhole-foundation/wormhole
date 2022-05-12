import { useEffect, useState } from "react";
import axios from "axios";
import {
  DataWrapper,
  errorDataWrapper,
  fetchDataWrapper,
  receiveDataWrapper,
} from "../store/helpers";
import { TOTAL_TRANSACTIONS_WORMHOLE } from "../utils/consts";

export interface Totals {
  TotalCount: { [chainId: string]: number };
  DailyTotals: {
    // "2021-08-22": { "*": 0 },
    [date: string]: { [groupByKey: string]: number };
  };
}

const useTransactionTotals = () => {
  const [totals, setTotals] = useState<DataWrapper<Totals>>(fetchDataWrapper());

  useEffect(() => {
    let cancelled = false;
    axios
      .get<Totals>(TOTAL_TRANSACTIONS_WORMHOLE)
      .then((response) => {
        if (!cancelled) {
          setTotals(receiveDataWrapper(response.data));
        }
      })
      .catch((error) => {
        if (!cancelled) {
          setTotals(errorDataWrapper(error));
          console.log(error);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return totals;
};

export default useTransactionTotals;
