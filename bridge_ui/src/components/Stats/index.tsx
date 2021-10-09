import {
  CircularProgress,
  Container,
  makeStyles,
  Paper,
  Typography,
} from "@material-ui/core";
import { useMemo } from "react";
import useTVL from "../../hooks/useTVL";
import { COLORS } from "../../muiTheme";
import { CHAINS } from "../../utils/consts";
import SmartAddress from "../SmartAddress";
import MuiReactTable from "./tableComponents/MuiReactTable";
import numeral from "numeral";
import clsx from "clsx";
import { ChainId } from "@certusone/wormhole-sdk";

const useStyles = makeStyles((theme) => ({
  logoPositioner: {
    width: "30px",
    maxWidth: "30px",
    marginRight: theme.spacing(1),
  },
  logo: {
    maxWidth: "100%",
  },
  tokenContainer: {
    display: "flex",
    justifyContent: "flex-start",
    alignItems: "center",
  },
  mainPaper: {
    backgroundColor: COLORS.nearBlackWithMinorTransparency,
    textAlign: "center",
    padding: "2rem",
    "& > h, p ": {
      margin: ".5rem",
    },
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
}));

const StatsRoot: React.FC<any> = () => {
  const classes = useStyles();
  const tvl = useTVL();
  const sortTokens = useMemo(() => {
    return (rowA: any, rowB: any) => {
      if (rowA.original.symbol && !rowB.original.symbol) {
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
        accessor: (value: any) => ({
          chainId: CHAINS.find((x) => x.name === value.originChain)?.id,
          symbol: value.symbol,
          name: value.name,
          logo: value.logo,
          assetAddress: value.assetAddress,
        }),
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
              chainId={
                CHAINS.find((x) => x.name === value.originChain)?.id as ChainId
              }
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
        Cell: (value: any) =>
          value.row?.original?.amount !== undefined
            ? numeral(value.row?.original?.amount).format("0,0.00")
            : "",
      },
      {
        Header: "Total Value (USD)",
        accessor: "totalValue",
        align: "right",
        Cell: (value: any) =>
          value.row?.original?.totalValue !== undefined
            ? numeral(value.row?.original?.totalValue).format("0.0 a")
            : "",
      },
      {
        Header: "Unit Price (USD)",
        accessor: "quotePrice",
        align: "right",
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
      return numeral(sum).format("0 a").toUpperCase();
    }
  }, [tvl.data]);

  return (
    <Container maxWidth="lg">
      <Paper className={classes.mainPaper}>
        {tvl.isFetching ? (
          <CircularProgress />
        ) : (
          <>
            <div className={classes.flexBox}>
              <div className={classes.explainerContainer}>
                <Typography variant="h5">Total Value Locked</Typography>
                <Typography variant="subtitle2" color="textSecondary">
                  These assets are currently locked by the Token Bridge
                  contracts.
                </Typography>
              </div>
              <div className={classes.grower} />
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
            </div>
            <MuiReactTable
              columns={tvlColumns}
              data={tvl.data}
              skipPageReset={false}
            />
          </>
        )}
      </Paper>
    </Container>
  );
};

export default StatsRoot;
