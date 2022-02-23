import { Typography } from "@mui/material";
import React, { useEffect, useState } from "react";
import { chainIDStrings } from "../../utils/consts";
import { amountFormatter } from "../../utils/explorer";
import {
  NotionalTransferred,
  NotionalTransferredToCumulative,
  Totals,
} from "./ExplorerStats";

interface ChainOverviewCardProps {
  dataKey: keyof typeof chainIDStrings;
  totals?: Totals;
  notionalTransferred?: NotionalTransferred;
  notionalTransferredToCumulative?: NotionalTransferredToCumulative;
}

const ChainOverviewCard: React.FC<ChainOverviewCardProps> = ({
  dataKey,
  totals,
  notionalTransferred,
  notionalTransferredToCumulative,
}) => {
  const [totalCount, setTotalColunt] = useState<number>();
  const [animate, setAnimate] = useState<boolean>(false);

  useEffect(() => {
    // hold values from props in state, so that we can detect changes and add animation class
    setTotalColunt(totals?.TotalCount[dataKey]);

    let timeout: NodeJS.Timeout;
    if (
      totals?.LastDayCount[dataKey] &&
      totalCount !== totals?.LastDayCount[dataKey]
    ) {
      setAnimate(true);
      timeout = setTimeout(() => {
        setAnimate(false);
      }, 2000);
    }
    return function cleanup() {
      if (timeout) {
        clearTimeout(timeout);
      }
    };
  }, [
    totals?.TotalCount[dataKey],
    totals?.LastDayCount[dataKey],
    dataKey,
    totalCount,
  ]);

  const centerStyles: any = {
    display: "flex",
    justifyContent: "flex-start",
    alignItems: "center",
    flexDirection: "column",
  };
  return (
    <>
      <div style={{ ...centerStyles, gap: 8 }}>
        {notionalTransferredToCumulative &&
          notionalTransferredToCumulative.AllTime && (
            <div style={centerStyles}>
              <div>
                <Typography
                  variant="h5"
                  className={animate ? "highlight-new-val" : ""}
                >
                  $
                  {amountFormatter(
                    notionalTransferredToCumulative.AllTime[dataKey]["*"]
                  )}
                </Typography>
              </div>
              <div style={{ marginTop: -10 }}>
                <Typography variant="caption">received</Typography>
              </div>
            </div>
          )}
        {notionalTransferred &&
        notionalTransferred.WithinPeriod &&
        dataKey in notionalTransferred.WithinPeriod &&
        "*" in notionalTransferred.WithinPeriod[dataKey] &&
        "*" in notionalTransferred.WithinPeriod[dataKey]["*"] &&
        notionalTransferred.WithinPeriod[dataKey]["*"]["*"] > 0 ? (
          <div style={centerStyles}>
            <div>
              <Typography
                variant="h5"
                className={animate ? "highlight-new-val" : ""}
              >
                {notionalTransferred.WithinPeriod[dataKey]["*"]["*"]
                  ? "$" +
                    amountFormatter(
                      notionalTransferred.WithinPeriod[dataKey]["*"]["*"]
                    )
                  : "..."}
              </Typography>
            </div>
            <div style={{ marginTop: -10 }}>
              <Typography variant="caption">sent</Typography>
            </div>
          </div>
        ) : (
          <div style={centerStyles}>
            <div style={{ marginTop: -10 }}>
              <Typography variant="body1">
                amount sent
                <br />
                coming soon
              </Typography>
            </div>
          </div>
        )}
        {!!totalCount && (
          <div style={centerStyles}>
            <div>
              <Typography
                variant="h5"
                className={animate ? "highlight-new-val" : ""}
              >
                {amountFormatter(totalCount)}
              </Typography>
            </div>
            <div style={{ marginTop: -10 }}>
              <Typography variant="caption"> messages </Typography>
            </div>
          </div>
        )}
      </div>

      {totalCount === 0 && <Typography variant="h6">coming soon</Typography>}
    </>
  );
};

export default ChainOverviewCard;
