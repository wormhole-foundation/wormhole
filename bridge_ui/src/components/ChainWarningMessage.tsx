import { ChainId } from "@certusone/wormhole-sdk";
import { Link, makeStyles, Typography } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { useMemo } from "react";
import { CHAIN_CONFIG_MAP } from "../config";

const useStyles = makeStyles((theme) => ({
  alert: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
}));

export default function ChainWarningMessage({ chainId }: { chainId: ChainId }) {
  const classes = useStyles();

  const warningMessage = useMemo(() => {
    return CHAIN_CONFIG_MAP[chainId]?.warningMessage;
  }, [chainId]);

  if (warningMessage === undefined) {
    return null;
  }

  return (
    <Alert variant="outlined" severity="warning" className={classes.alert}>
      {warningMessage.text}
      {warningMessage.link ? (
        <Typography component="div">
          <Link href={warningMessage.link.url} target="_blank" rel="noreferrer">
            {warningMessage.link.text}
          </Link>
        </Typography>
      ) : null}
    </Alert>
  );
}
