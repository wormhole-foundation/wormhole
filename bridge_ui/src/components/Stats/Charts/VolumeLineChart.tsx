import { ChainId } from "@certusone/wormhole-sdk";
import { useTheme, useMediaQuery } from "@material-ui/core";
import {
  Legend,
  ResponsiveContainer,
  LineChart,
  XAxis,
  YAxis,
  Line,
  Tooltip,
} from "recharts";
import { COLOR_BY_CHAIN_ID, getChainShortName } from "../../../utils/consts";
import MultiChainTooltip from "./MultiChainTooltip";
import { TimeFrame } from "./TimeFrame";
import { formatTVL, renderLegendText, TransferChartData } from "./utils";

const VolumeLineChart = ({
  transferData,
  timeFrame,
  chains,
}: {
  transferData: TransferChartData[];
  timeFrame: TimeFrame;
  chains: ChainId[];
}) => {
  const theme = useTheme();
  const isXSmall = useMediaQuery(theme.breakpoints.down("xs"));

  return (
    <ResponsiveContainer height={452}>
      <LineChart data={transferData}>
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
          tickFormatter={formatTVL}
          tick={{ fill: "white" }}
          axisLine={false}
          tickLine={false}
        />
        <Tooltip
          content={
            <MultiChainTooltip
              title="Multiple Chains"
              valueFormatter={formatTVL}
            />
          }
        />
        {chains.map((chainId) => (
          <Line
            dataKey={`transferredByChain.${chainId}`}
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

export default VolumeLineChart;
