import { Link, makeStyles, Typography } from "@material-ui/core";
import { Alert } from "@material-ui/lab";

const useStyles = makeStyles((theme) => ({
  alert: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
}));

export default function PolygonNetworkDownWarning() {
  const classes = useStyles();

  return (
    <Alert variant="outlined" severity="warning" className={classes.alert}>
      Polygon is experiencing downtime. As a precautionary measure, Wormhole
      network and portal have temporarily paused Polygon support until the issue
      is resolved.
      <Typography component="div">
        <Link
          href="https://twitter.com/0xPolygonDevs/status/1501944974933303305"
          target="_blank"
          rel="noopener noreferrer"
        >
          Link to @0xPolygonDevs tweet
        </Link>
      </Typography>
    </Alert>
  );
}
