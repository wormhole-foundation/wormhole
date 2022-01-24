import { makeStyles } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { Connection } from "@solana/web3.js";
import numeral from "numeral";
import { useEffect, useState } from "react";
import { SOLANA_HOST } from "../utils/consts";

const useStyles = makeStyles((theme) => ({
  alert: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
}));

export default function SolanaTPSWarning() {
  const classes = useStyles();
  const [tps, setTps] = useState<number | null>(null);
  useEffect(() => {
    let cancelled = false;
    let interval = setInterval(() => {
      (async () => {
        try {
          const connection = new Connection(SOLANA_HOST);
          const samples = await connection.getRecentPerformanceSamples(1);
          if (samples.length >= 1) {
            let short = samples
              .filter((sample) => sample.numTransactions !== 0)
              .map(
                (sample) => sample.numTransactions / sample.samplePeriodSecs
              );
            const avgTps = short[0];
            if (!cancelled) {
              setTps(avgTps);
            }
          }
        } catch (e) {}
      })();
    }, 5000);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, []);
  return tps !== null && tps < 1500 ? (
    <Alert
      variant="outlined"
      severity="warning"
      className={classes.alert}
    >{`WARNING! The Solana Transactions Per Second (TPS) is below 1500. This is a sign of network congestion. Proceed with caution as you may have difficulty submitting transactions and the guardians may have difficulty witnessing them (this could lead to processing delays). Current TPS: ${numeral(
      tps
    ).format("0,0")}`}</Alert>
  ) : null;
}
