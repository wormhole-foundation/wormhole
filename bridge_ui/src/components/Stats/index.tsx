import { BigNumber } from "@ethersproject/bignumber";
import { formatUnits, parseUnits } from "@ethersproject/units";
import {
  CircularProgress,
  Container,
  makeStyles,
  Paper,
  Typography,
} from "@material-ui/core";
import clsx from "clsx";
import numeral from "numeral";
import { useMemo } from "react";
import useTVL from "../../hooks/useTVL";
import { COLORS } from "../../muiTheme";
import SmartAddress from "../SmartAddress";
import { balancePretty } from "../TokenSelectors/TokenPicker";
import CustodyAddresses from "./CustodyAddresses";
import NFTStats from "./NFTStats";
import MuiReactTable from "./tableComponents/MuiReactTable";
import TransactionMetrics from "./TransactionMetrics";

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
  mainPaper: {
    backgroundColor: COLORS.nearBlackWithMinorTransparency,
    padding: "2rem",
    "& > h, & > p ": {
      margin: ".5rem",
    },
    marginBottom: theme.spacing(2),
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
  explainerContainer: {},
  totalContainer: {
    display: "flex",
    alignItems: "flex-end",
    paddingBottom: 1, // line up with left text bottom
    [theme.breakpoints.down("sm")]: {
      marginTop: theme.spacing(1),
    },
  },
  totalValue: {
    marginLeft: theme.spacing(0.5),
    marginBottom: "-.125em", // line up number with label
  },
  alignCenter: {
    margin: "0 auto",
    display: "block",
  },
}));

const StatsRoot: React.FC<any> = () => {
  const classes = useStyles();
  const tvl = useTVL();

  const sortTokens = useMemo(() => {
    return (rowA: any, rowB: any) => {
      if (rowA.isGrouped && rowB.isGrouped) {
        return rowA.values.assetAddress > rowB.values.assetAddress ? 1 : -1;
      } else if (rowA.isGrouped && !rowB.isGrouped) {
        return 1;
      } else if (!rowA.isGrouped && rowB.isGrouped) {
        return -1;
      } else if (rowA.original.symbol && !rowB.original.symbol) {
        return 1;
      } else if (rowB.original.symbol && !rowA.original.symbol) {
        return -1;
      } else if (rowA.original.symbol && rowB.original.symbol) {
        return rowA.original.symbol > rowB.original.symbol ? 1 : -1;
      } else {
        return rowA.original.assetAddress > rowB.original.assetAddress ? 1 : -1;
      }
    };
  }, []);
  const tvlColumns = useMemo(() => {
    return [
      {
        Header: "Token",
        id: "assetAddress",
        sortType: sortTokens,
        disableGroupBy: true,
        accessor: (value: any) => ({
          chainId: value.originChainId,
          symbol: value.symbol,
          name: value.name,
          logo: value.logo,
          assetAddress: value.assetAddress,
        }),
        aggregate: (leafValues: any) => leafValues.length,
        Aggregated: ({ value }: { value: any }) =>
          `${value} Token${value === 1 ? "" : "s"}`,
        Cell: (value: any) => (
          <div className={classes.tokenContainer}>
            <div className={classes.logoPositioner}>
              {value.row?.original?.logo ? (
                <img
                  src={value.row?.original?.logo}
                  alt=""
                  className={classes.logo}
                />
              ) : null}
            </div>
            <SmartAddress
              chainId={value.row?.original?.originChainId}
              address={value.row?.original?.assetAddress}
              symbol={value.row?.original?.symbol}
              tokenName={value.row?.original?.name}
            />
          </div>
        ),
      },
      { Header: "Chain", accessor: "originChain" },
      {
        Header: "Amount",
        accessor: "amount",
        align: "right",
        disableGroupBy: true,
        Cell: (value: any) =>
          value.row?.original?.amount !== undefined
            ? numeral(value.row?.original?.amount).format("0,0.00")
            : "",
      },
      {
        Header: "Total Value (USD)",
        id: "totalValue",
        accessor: "totalValue",
        align: "right",
        disableGroupBy: true,
        aggregate: (leafValues: any) =>
          balancePretty(
            formatUnits(
              leafValues.reduce(
                (p: BigNumber, v: number | null | undefined) =>
                  v ? p.add(parseUnits(v.toFixed(18).toString(), 18)) : p,
                BigNumber.from(0)
              ),
              18
            )
          ),
        Aggregated: ({ value }: { value: any }) => value,
        Cell: (value: any) =>
          value.row?.original?.totalValue !== undefined
            ? numeral(value.row?.original?.totalValue).format("0.0 a")
            : "",
      },
      {
        Header: "Unit Price (USD)",
        accessor: "quotePrice",
        align: "right",
        disableGroupBy: true,
        Cell: (value: any) =>
          value.row?.original?.quotePrice !== undefined
            ? numeral(value.row?.original?.quotePrice).format("0,0.00")
            : "",
      },
    ];
  }, [
    classes.logo,
    classes.tokenContainer,
    classes.logoPositioner,
    sortTokens,
  ]);
  const tvlString = useMemo(() => {
    if (!tvl.data) {
      return "";
    } else {
      let sum = 0;
      tvl.data.forEach((val) => {
        if (val.totalValue) sum += val.totalValue;
      });
      return numeral(sum)
        .format(sum >= 1000000000 ? "0.000 a" : "0 a")
        .toUpperCase();
    }
  }, [tvl.data]);

  return (
    <Container maxWidth="lg">
      <Paper className={classes.mainPaper}>
        <>
          <div className={classes.flexBox}>
            <div className={classes.explainerContainer}>
              <Typography variant="h5">Total Value Locked</Typography>
              <Typography variant="subtitle2" color="textSecondary">
                These assets are currently locked by the Token Bridge contracts.
              </Typography>
            </div>
            <div className={classes.grower} />
            {!tvl.isFetching ? (
              <div
                className={clsx(
                  classes.explainerContainer,
                  classes.totalContainer
                )}
              >
                <Typography
                  variant="body2"
                  color="textSecondary"
                  component="div"
                  noWrap
                >
                  {"Total (USD)"}
                </Typography>
                <Typography
                  variant="h3"
                  component="div"
                  noWrap
                  className={classes.totalValue}
                >
                  {tvlString}
                </Typography>
              </div>
            ) : null}
          </div>
          {!tvl.isFetching ? (
            <MuiReactTable
              columns={tvlColumns}
              data={tvl.data}
              skipPageReset={false}
              initialState={{ sortBy: [{ id: "totalValue", desc: true }] }}
            />
          ) : (
            <CircularProgress className={classes.alignCenter} />
          )}
        </>
      </Paper>
      <Paper className={classes.mainPaper}>
        <TransactionMetrics />
      </Paper>
      <Paper className={classes.mainPaper}>
        <CustodyAddresses />
      </Paper>
      <Paper className={classes.mainPaper}>
        <NFTStats />
      </Paper>
    </Container>
  );
};

export default StatsRoot;
