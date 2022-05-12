import { NotionalTVLCumulative } from "../../../hooks/useCumulativeTVL";
import { NotionalTransferredFrom } from "../../../hooks/useNotionalTransferred";
import { TimeFrame } from "./TimeFrame";
import { DateTime } from "luxon";
import { Totals } from "../../../hooks/useTransactionTotals";
import {
  ChainInfo,
  CHAINS_BY_ID,
  VAA_EMITTER_ADDRESSES,
} from "../../../utils/consts";
import { NotionalTVL } from "../../../hooks/useTVL";
import { ChainId } from "@certusone/wormhole-sdk";

export const formatTVL = (tvl: number) => {
  const [divisor, unit, fractionDigits] =
    tvl < 1e3
      ? [1, "", 0]
      : tvl < 1e6
      ? [1e3, "K", 0]
      : tvl < 1e9
      ? [1e6, "M", 0]
      : [1e9, "B", 2];
  return `$${(tvl / divisor).toFixed(fractionDigits)} ${unit}`;
};

export const formatDate = (date: Date) => {
  return date.toLocaleString("en-US", {
    day: "numeric",
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  });
};

export const formatTickDay = (date: Date) => {
  return date.toLocaleString("en-US", {
    day: "numeric",
    month: "short",
    year: "numeric",
    timeZone: "UTC",
  });
};

export const formatTickMonth = (date: Date) => {
  return date.toLocaleString("en-US", {
    month: "short",
    year: "numeric",
    timeZone: "UTC",
  });
};

export const formatTransactionCount = (transactionCount: number) => {
  return transactionCount.toLocaleString("en-US");
};

export const renderLegendText = (value: any) => {
  return <span style={{ color: "white", margin: "8px" }}>{value}</span>;
};

export const getStartDate = (timeFrame: TimeFrame) => {
  return timeFrame.duration
    ? DateTime.now().toUTC().minus(timeFrame.duration).toJSDate()
    : undefined;
};

export interface CumulativeTVLChartData {
  date: Date;
  totalTVL: number;
  tvlByChain: {
    [chainId: string]: number;
  };
}

export const createCumulativeTVLChartData = (
  cumulativeTVL: NotionalTVLCumulative,
  timeFrame: TimeFrame
) => {
  const startDate = getStartDate(timeFrame);
  return Object.entries(cumulativeTVL.DailyLocked)
    .reduce<CumulativeTVLChartData[]>(
      (chartData, [dateString, chainsAssets]) => {
        const date = new Date(dateString);
        if (!startDate || date >= startDate) {
          const data: CumulativeTVLChartData = {
            date: date,
            totalTVL: 0,
            tvlByChain: {},
          };
          Object.entries(chainsAssets).forEach(([chainId, lockedAssets]) => {
            const notional = lockedAssets["*"].Notional;
            if (chainId === "*") {
              data.totalTVL = notional;
            } else {
              data.tvlByChain[chainId] = notional;
            }
          });
          chartData.push(data);
        }
        return chartData;
      },
      []
    )
    .sort((a, z) => a.date.getTime() - z.date.getTime());
};

export interface TransferChartData {
  date: Date;
  totalTransferred: number;
  transferredByChain: {
    [chainId: string]: number;
  };
}

export const createTransferChartData = (
  notionalTransferredFrom: NotionalTransferredFrom,
  timeFrame: TimeFrame
) => {
  const startDate = getStartDate(timeFrame);
  return Object.keys(notionalTransferredFrom.Daily)
    .sort()
    .reduce<TransferChartData[]>((chartData, dateString) => {
      const transferFromData = notionalTransferredFrom.Daily[dateString];
      const data: TransferChartData = {
        date: new Date(dateString),
        totalTransferred: 0,
        transferredByChain: {},
      };
      Object.entries(transferFromData).forEach(([chainId, amount]) => {
        if (chainId === "*") {
          data.totalTransferred = amount;
        } else {
          data.transferredByChain[chainId] = amount;
        }
      });
      chartData.push(data);
      return chartData;
    }, [])
    .filter((value) => !startDate || startDate <= value.date);
};

export interface TransactionData {
  date: Date;
  totalTransactions: number;
  transactionsByChain: {
    [chainId: string]: number;
  };
}

export const createTransactionData = (totals: Totals, timeFrame: TimeFrame) => {
  const startDate = getStartDate(timeFrame);
  return Object.keys(totals.DailyTotals)
    .sort()
    .reduce<TransactionData[]>((chartData, dateString) => {
      const groupByKeys = totals.DailyTotals[dateString];
      const data: TransactionData = {
        date: new Date(dateString),
        totalTransactions: 0,
        transactionsByChain: {},
      };
      VAA_EMITTER_ADDRESSES.forEach((address) => {
        const count = groupByKeys[address] || 0;
        data.totalTransactions += count;
        const chainId = address.slice(0, address.indexOf(":"));
        if (data.transactionsByChain[chainId] === undefined) {
          data.transactionsByChain[chainId] = 0;
        }
        data.transactionsByChain[chainId] += count;
      });
      chartData.push(data);
      return chartData;
    }, [])
    .filter((value) => !startDate || startDate <= value.date);
};

export interface ChainTVLChartData {
  chainInfo: ChainInfo;
  tvl: number;
  tvlRatio: number;
}

export const createChainTVLChartData = (tvl: NotionalTVL) => {
  let maxTVL = 0;
  const chainTVLs = Object.entries(tvl.AllTime)
    .reduce<ChainTVLChartData[]>((chartData, [chainId, assets]) => {
      const chainInfo = CHAINS_BY_ID[+chainId as ChainId];
      if (chainInfo !== undefined) {
        const tvl = assets["*"].Notional;
        chartData.push({
          chainInfo: chainInfo,
          tvl: tvl,
          tvlRatio: 0,
        });
        maxTVL = Math.max(maxTVL, tvl);
      }
      return chartData;
    }, [])
    .sort((a, z) => z.tvl - a.tvl);
  if (maxTVL > 0) {
    chainTVLs.forEach((chainTVL) => {
      chainTVL.tvlRatio = (chainTVL.tvl / maxTVL) * 100;
    });
  }
  return chainTVLs;
};
