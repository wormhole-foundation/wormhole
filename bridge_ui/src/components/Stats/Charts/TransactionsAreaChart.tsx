import { useTheme, useMediaQuery } from "@material-ui/core";
import { useCallback } from "react";
import {
  Area,
  AreaChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import CustomTooltip from "./CustomTooltip";
import { TimeFrame } from "./TimeFrame";
import { formatTransactionCount, TransactionData } from "./utils";

const TransactionsAreaChart = ({
  transactionData,
  timeFrame,
}: {
  transactionData: TransactionData[];
  timeFrame: TimeFrame;
}) => {
  const formatValue = useCallback((value: number) => {
    return `${formatTransactionCount(value)} transactions`;
  }, []);

  const theme = useTheme();
  const isXSmall = useMediaQuery(theme.breakpoints.down("xs"));

  return (
    <ResponsiveContainer height={452}>
      <AreaChart data={transactionData}>
        <XAxis
          dataKey="date"
          tickFormatter={timeFrame.tickFormatter}
          tick={{ fill: "white" }}
          interval={!isXSmall ? timeFrame.interval : undefined}
          axisLine={false}
          tickLine={false}
          dy={16}
          padding={{ right: 32 }}
        />
        <YAxis
          tickFormatter={formatTransactionCount}
          tick={{ fill: "white" }}
          axisLine={false}
          tickLine={false}
        />
        <Tooltip
          content={
            <CustomTooltip title="All chains" valueFormatter={formatValue} />
          }
        />
        <defs>
          <linearGradient id="gradient" gradientTransform="rotate(100)">
            <stop offset="0%" stopColor="#FF2B57" />
            <stop offset="100%" stopColor="#5EA1EC" />
          </linearGradient>
        </defs>
        <Area
          dataKey="totalTransactions"
          stroke="#405BBC"
          fill="url(#gradient)"
        />
      </AreaChart>
    </ResponsiveContainer>
  );
};

export default TransactionsAreaChart;
