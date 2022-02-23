import React, { useEffect, useState } from "react";
import { Totals } from "./ExplorerStats";
import { ResponsiveBar, BarDatum } from "@nivo/bar";

import {
  chainColors,
  chainIdColors,
  makeDate,
  makeGroupName,
} from "../../utils/explorer";
import { useNetworkContext } from "../../contexts/NetworkContext";
import { chainEnums } from "../../utils/consts";
import { Typography } from "@mui/material";

interface DailyCountProps {
  dailyCount: Totals["DailyTotals"];
  showBottomLedgend?: boolean;
}

const DailyCountBarChart = (props: DailyCountProps) => {
  const { activeNetwork } = useNetworkContext();
  const [data, setData] = useState<Array<BarDatum>>([]);

  useEffect(() => {
    const datum = Object.keys(props.dailyCount).reduce<Array<BarDatum>>(
      (accum, date) => {
        const chains = props.dailyCount[date];
        return [
          ...accum,
          Object.keys(chains).reduce<BarDatum>(
            (subAccum, chain) => {
              const group = makeGroupName(chain, activeNetwork);
              return {
                ...subAccum,
                [group]: chains[chain],
              };
            },
            { date: date }
          ),
        ];
      },
      []
    );

    setData(datum);
  }, [props.dailyCount, activeNetwork]);

  const keys = chainEnums.slice(1);
  const today = new Date().toISOString().slice(0, 10);

  return (
    <div style={{ height: 400, minWidth: 360, flex: "1", marginBottom: 40 }}>
      <Typography variant="subtitle1">Messages/Day</Typography>
      <ResponsiveBar
        theme={{ textColor: "rgba(255, 255, 255, 0.85)" }}
        data={data}
        keys={keys}
        colors={chainIdColors.slice(1)}
        groupMode="stacked"
        indexBy="date"
        margin={{
          top: 10,
          right: 0,
          bottom: props.showBottomLedgend ? 80 : 24,
          left: 40,
        }}
        padding={0.3}
        valueScale={{ type: "linear" }}
        indexScale={{ type: "band", round: true }}
        borderColor={{ from: "color", modifiers: [["darker", 1.6]] }}
        axisTop={null}
        axisRight={null}
        axisBottom={{
          format: (value) => {
            if (value === today) {
              return "today";
            }
            return makeDate(value);
          },
        }}
        labelSkipWidth={12}
        labelSkipHeight={12}
        labelTextColor={{ from: "color", modifiers: [["darker", 3]] }}
        tooltip={(data) => {
          let { id, value, indexValue } = data;

          return (
            <div
              style={{
                background: "#827db8",
                borderRadius: "14px",
                padding: "9px 12px",
                border: "1px solid rgba(255, 255, 255, 0.85)",
                color: "rgba(255, 255, 255, 0.85)",
                fontSize: 14,
              }}
            >
              <Typography
                variant="subtitle1"
                style={{ color: "rgba(255, 255, 255, 0.85)" }}
              >
                {id} - {makeDate(String(indexValue))}
              </Typography>
              <div
                style={{
                  display: "flex",
                  padding: "3px 0",
                  justifyContent: "flex-end",
                }}
              >
                <span>{value} messages</span>
              </div>
            </div>
          );
        }}
      />
      {props.showBottomLedgend && (
        <div
          style={{
            display: "flex",
            justifyContent: "space-evenly",
            width: "100%",
          }}
        >
          {chainEnums.slice(1).map((chain, index) => (
            <div key={chain} style={{ display: "flex", alignItems: "center" }}>
              <div
                style={{
                  background: chainColors[String(index + 1)],
                  height: 16,
                  width: 16,
                  display: "inline-block",
                }}
              />
              &nbsp;
              <span>{chain}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default DailyCountBarChart;
