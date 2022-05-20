import {
  AreaChart,
  Area,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { formatTVL, createCumulativeTVLChartData } from "./utils";
import { NotionalTVLCumulative } from "../../../hooks/useCumulativeTVL";
import { useMemo } from "react";
import { TimeFrame } from "./TimeFrame";
import CustomTooltip from "./CustomTooltip";
import { useTheme, useMediaQuery } from "@material-ui/core";

const TVLAreaChart = ({
  cumulativeTVL,
  timeFrame,
}: {
  cumulativeTVL: NotionalTVLCumulative;
  timeFrame: TimeFrame;
}) => {
  const data = useMemo(() => {
    return createCumulativeTVLChartData(cumulativeTVL, timeFrame);
  }, [cumulativeTVL, timeFrame]);

  const theme = useTheme();
  const isXSmall = useMediaQuery(theme.breakpoints.down("xs"));

  return (
    <ResponsiveContainer height={452}>
      <AreaChart data={data}>
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
          content={<CustomTooltip title="TVL" valueFormatter={formatTVL} />}
        />
        <defs>
          <linearGradient id="gradient" gradientTransform="rotate(100)">
            <stop offset="0%" stopColor="#FF2B57" />
            <stop offset="100%" stopColor="#5EA1EC" />
          </linearGradient>
        </defs>
        <Area dataKey="totalTVL" fill="url(#gradient)" stroke="#405BBC" />
      </AreaChart>
    </ResponsiveContainer>
  );
};

export default TVLAreaChart;
