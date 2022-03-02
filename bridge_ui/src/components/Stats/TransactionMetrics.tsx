import {
  CircularProgress,
  Link,
  makeStyles,
  Paper,
  Typography,
} from "@material-ui/core";
import clsx from "clsx";
import numeral from "numeral";
import useTransactionCount from "../../hooks/useTransactionCount";
import { COLORS } from "../../muiTheme";
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
    marginBottom: theme.spacing(4),
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
    paddingLeft: theme.spacing(1),
    paddingRight: theme.spacing(1),
    textAlign: "center",
    marginBottom: theme.spacing(2),
  },
  totalValue: {
    fontWeight: 600,
    fontFamily: "Suisse BP Intl, sans-serif",
  },
  typog: {
    marginTop: theme.spacing(3),
  },
  mainPaper: {
    backgroundColor: COLORS.whiteWithTransparency,
    padding: "2rem",
    "& > h, & > p ": {
      margin: ".5rem",
    },
    marginBottom: theme.spacing(8),
  },
}));

const TransactionMetrics: React.FC<any> = () => {
  const transactionCount = useTransactionCount();
  const classes = useStyles();
  const isFetching = transactionCount.isFetching;

  const header = (
    <div className={classes.flexBox}>
      <div>
        <Typography variant="h4">Transaction Count</Typography>
        <Typography variant="subtitle1" color="textSecondary">
          This is how many transactions the Token Bridge has processed.
        </Typography>
      </div>
      <div className={classes.grower} />
    </div>
  );

  const content = (
    <div className={classes.totalsBox}>
      <div className={classes.totalContainer}>
        <Typography variant="subtitle2" component="div" noWrap>
          {"Last 48 Hours"}
        </Typography>
        <Typography
          variant="h2"
          component="div"
          noWrap
          className={classes.totalValue}
        >
          {numeral(transactionCount.data?.total48h || 0).format("0,0")}
        </Typography>
      </div>
      <div className={classes.totalContainer}>
        <Typography variant="subtitle2" component="div" noWrap>
          {"All Time"}
        </Typography>
        <Typography
          variant="h2"
          component="div"
          noWrap
          className={classes.totalValue}
        >
          {numeral(transactionCount.data?.totalAllTime || 0).format("0,0")}
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
      <Paper className={classes.mainPaper}>
        {isFetching ? (
          <CircularProgress className={classes.alignCenter} />
        ) : (
          <>
            {content}
            {networkExplorer}
          </>
        )}
      </Paper>
    </>
  );
};

export default TransactionMetrics;
