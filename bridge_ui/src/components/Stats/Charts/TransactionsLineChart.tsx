import { ChainId } from "@certusone/wormhole-sdk";
import { useTheme, useMediaQuery } from "@material-ui/core";
import { useCallback } from "react";
import {
  ResponsiveContainer,
  LineChart,
  XAxis,
  YAxis,
  Line,
  Legend,
  Tooltip,
} from "recharts";
import { COLOR_BY_CHAIN_ID, getChainShortName } from "../../../utils/consts";
import MultiChainTooltip from "./MultiChainTooltip";
import { TimeFrame } from "./TimeFrame";
import {
  formatTransactionCount,
  renderLegendText,
  TransactionData,
} from "./utils";

const TransactionsLineChart = ({
  transactionData,
  timeFrame,
  chains,
}: {
  transactionData: TransactionData[];
  timeFrame: TimeFrame;
  chains: ChainId[];
}) => {
  const formatValue = useCallback((value: number) => {
    return `${formatTransactionCount(value)} transactions`;
  }, []);

  const theme = useTheme();
  const isXSmall = useMediaQuery(theme.breakpoints.down("xs"));

  return (
    <ResponsiveContainer height={452}>
      <LineChart data={transactionData}>
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
            <MultiChainTooltip
              title="Multiple Chains"
              valueFormatter={formatValue}
            />
          }
        />
        {chains.map((chainId) => (
          <Line
            dataKey={`transactionsByChain.${chainId}`}
            name={getChainShortName(chainId)}
            stroke={COLOR_BY_CHAIN_ID[chainId]}
            strokeWidth="4"
            dot={false}
            key={chainId}
          />
        ))}
        <Legend
          iconType="square"
          iconSize={32}
          formatter={renderLegendText}
          wrapperStyle={{ paddingTop: 24 }}
        />
      </LineChart>
    </ResponsiveContainer>
  );
};

export default TransactionsLineChart;
