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
      Polygon is currently experiencing partial downtime.
      As a precautionary measure, Wormhole Network and Portal have paused Polygon
      support until the network has been fully restored.
      <Typography component="div">
        <Link
          href="https://twitter.com/0xPolygonDevs"
          target="_blank"
          rel="noopener noreferrer"
        >
          Follow @0xPolygonDevs for updates
        </Link>
      </Typography>
    </Alert>
  );
}
