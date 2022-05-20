import { ChainId } from "@certusone/wormhole-sdk";
import { useTheme, useMediaQuery } from "@material-ui/core";
import { useMemo } from "react";
import {
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { NotionalTVLCumulative } from "../../../hooks/useCumulativeTVL";
import { COLOR_BY_CHAIN_ID, getChainShortName } from "../../../utils/consts";
import MultiChainTooltip from "./MultiChainTooltip";
import { TimeFrame } from "./TimeFrame";
import {
  formatTVL,
  createCumulativeTVLChartData,
  renderLegendText,
} from "./utils";

const TVLLineChart = ({
  cumulativeTVL,
  timeFrame,
  selectedChains,
}: {
  cumulativeTVL: NotionalTVLCumulative;
  timeFrame: TimeFrame;
  selectedChains: ChainId[];
}) => {
  const data = useMemo(() => {
    return createCumulativeTVLChartData(cumulativeTVL, timeFrame);
  }, [cumulativeTVL, timeFrame]);

  const theme = useTheme();
  const isXSmall = useMediaQuery(theme.breakpoints.down("xs"));

  return (
    <ResponsiveContainer height={452}>
      <LineChart data={data}>
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
        {selectedChains.map((chainId) => (
          <Line
            dataKey={`tvlByChain.${chainId}`}
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

export default TVLLineChart;
