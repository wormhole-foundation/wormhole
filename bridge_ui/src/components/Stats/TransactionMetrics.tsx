import {
  CircularProgress,
  Link,
  makeStyles,
  Typography,
} from "@material-ui/core";
import clsx from "clsx";
import useTransactionCount from "../../hooks/useTransactionCount";
import { WORMHOLE_EXPLORER_BASE } from "../../utils/consts";

const useStyles = makeStyles((theme) => ({
  logoPositioner: {
    height: "30px",
    width: "30px",
    maxWidth: "30px",
    marginRight: theme.spacing(1),
    display: "flex",
    alignItems: "center",
  },
  logo: {
    maxHeight: "100%",
    maxWidth: "100%",
  },
  tokenContainer: {
    display: "flex",
    justifyContent: "flex-start",
    alignItems: "center",
  },
  flexBox: {
    display: "flex",
    alignItems: "flex-end",
    marginBottom: theme.spacing(1),
    textAlign: "left",
    [theme.breakpoints.down("sm")]: {
      flexDirection: "column",
      alignItems: "unset",
    },
  },
  grower: {
    flexGrow: 1,
  },
  alignCenter: {
    margin: "0 auto",
    display: "block",
    textAlign: "center",
  },
  totalsBox: {
    display: "flex",
    flexWrap: "wrap",
    width: "100%",
    justifyContent: "space-evenly",
    alignItems: "center",
  },
  totalContainer: {
    display: "flex",
    alignItems: "flex-end",
    paddingLeft: theme.spacing(0.5),
    paddingRight: theme.spacing(0.5),
    paddingBottom: 1, // line up with left text bottom
    [theme.breakpoints.down("sm")]: {
      marginTop: theme.spacing(1),
    },
  },
  totalValue: {
    marginLeft: theme.spacing(0.5),
    marginBottom: "-.125em", // line up number with label
  },
  typog: {
    marginTop: theme.spacing(3),
  },
}));

const TransactionMetrics: React.FC<any> = () => {
  const transactionCount = useTransactionCount();
  const classes = useStyles();
  const isFetching = transactionCount.isFetching;

  const header = (
    <div className={classes.flexBox}>
      <div>
        <Typography variant="h5">Transaction Count</Typography>
        <Typography variant="subtitle2" color="textSecondary">
          This is how many transactions the Token Bridge has processed.
        </Typography>
      </div>
      <div className={classes.grower} />
    </div>
  );

  const content = (
    <div className={classes.totalsBox}>
      <div className={classes.totalContainer}>
        <Typography
          variant="body2"
          color="textSecondary"
          component="div"
          noWrap
        >
          {"Last 48 Hours"}
        </Typography>
        <Typography
          variant="h3"
          component="div"
          noWrap
          className={classes.totalValue}
        >
          {transactionCount.data?.total24h || "0"}
        </Typography>
      </div>
      <div className={classes.totalContainer}>
        <Typography
          variant="body2"
          color="textSecondary"
          component="div"
          noWrap
        >
          {"All Time"}
        </Typography>
        <Typography
          variant="h3"
          component="div"
          noWrap
          className={classes.totalValue}
        >
          {transactionCount.data?.totalAllTime || "0"}
        </Typography>
      </div>
    </div>
  );

  const networkExplorer = (
    <Typography
      variant="subtitle1"
      className={clsx(classes.alignCenter, classes.typog)}
    >
      To see metrics for the entire Wormhole Network (not just this bridge),
      check out the{" "}
      <Link href={WORMHOLE_EXPLORER_BASE} target="_blank">
        Wormhole Network Explorer
      </Link>
    </Typography>
  );

  return (
    <>
      {header}
      {isFetching ? (
        <CircularProgress className={classes.alignCenter} />
      ) : (
        <>
          {content}
          {networkExplorer}
        </>
      )}
    </>
  );
};

export default TransactionMetrics;
