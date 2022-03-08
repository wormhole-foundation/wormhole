import { makeStyles } from "@material-ui/core";
import { Alert } from "@material-ui/lab";

const useStyles = makeStyles((theme) => ({
  alert: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
}));

export default function OasisNetworkUpgradeWarning() {
  const classes = useStyles();

  return (
    <Alert variant="outlined" severity="warning" className={classes.alert}>
      Transfers from Oasis to other chains are currently unavailable due to a
      network software upgrade. Transfers from other chains to Oasis are
      unaffected.
    </Alert>
  );
}
