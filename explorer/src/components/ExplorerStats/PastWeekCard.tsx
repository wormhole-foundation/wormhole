import { Box, Card, Typography } from "@mui/material";
import React from "react";
import { chainEnums, chainIDs } from "../../utils/consts";
import { chainColors } from "../../utils/explorer";
import DailyCountBarChart from "./DailyCountBarChart";
import DailyNotionalBarChart from "./DailyNotionalBarChart";
import {
  NotionalTransferred,
  NotionalTransferredTo,
  Totals,
} from "./ExplorerStats";

interface PastWeekCardProps {
  title: string;
  messages: Totals;
  numDaysToShow: number;
  notionalTransferred?: NotionalTransferred;
  notionalTransferredTo: NotionalTransferredTo;
}

const PastWeekCard: React.FC<PastWeekCardProps> = ({
  title,
  messages,
  numDaysToShow,
  notionalTransferredTo,
}) => {
  const dates = [...Array(numDaysToShow)]
    .map((_, i) => {
      const d = new Date();
      d.setDate(d.getDate() - i);
      return d;
    })
    .map((d) => {
      const isoStr = d.toISOString();
      return isoStr.slice(0, 10);
    })
    .reverse();

  let messagesForPeriod = dates
    .filter((date) => messages && date in messages?.DailyTotals)
    .reduce<{ [date: string]: { [groupByKey: string]: number } }>(
      (accum, key) => ({ ...accum, [key]: messages.DailyTotals[key] }),
      Object()
    );

  let notionalTransferredToInPeriod = dates
    .filter((date) => date in notionalTransferredTo.Daily)
    .reduce<NotionalTransferredTo["Daily"]>(
      (accum, key) => ((accum[key] = notionalTransferredTo.Daily[key]), accum),
      Object()
    );

  return (
    <Card
      sx={{
        backgroundColor: "rgba(255,255,255,.07)",
        backgroundImage: "none",
        borderRadius: "28px",
        padding: "24px",
      }}
    >
      <Typography variant="h4" gutterBottom>
        {title}
      </Typography>
      <div
        style={{
          display: "flex",
          justifyContent: "space-evenly",
          alignItems: "center",
          flexWrap: "wrap",
          gap: 16,
          marginBottom: 10,
        }}
      >
        <DailyCountBarChart dailyCount={messagesForPeriod} />

        <DailyNotionalBarChart daily={notionalTransferredToInPeriod} />
      </div>

      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          justifyContent: "space-evenly",
          width: "100%",
        }}
      >
        {Object.values(chainIDs).map((chainId) => (
          <Box
            key={chainId}
            sx={{ display: "flex", alignItems: "center", mx: 1 }}
          >
            <div
              style={{
                background: chainColors[String(chainId)],
                height: 12,
                width: 12,
                display: "inline-block",
              }}
            />
            <div>&nbsp;{chainEnums[chainId]}</div>
          </Box>
        ))}
      </div>
    </Card>
  );
};

export default PastWeekCard;
