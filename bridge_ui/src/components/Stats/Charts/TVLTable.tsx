import { makeStyles } from "@material-ui/core";
import numeral from "numeral";
import { useMemo } from "react";
import { createTVLArray, NotionalTVL } from "../../../hooks/useTVL";
import { ChainInfo } from "../../../utils/consts";
import SmartAddress from "../../SmartAddress";
import MuiReactTable from "../tableComponents/MuiReactTable";
import { formatTVL } from "./utils";

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
}));

const TVLTable = ({
  chainInfo,
  tvl,
}: {
  chainInfo: ChainInfo;
  tvl: NotionalTVL;
}) => {
  const classes = useStyles();
  const chainTVL = useMemo(() => {
    return createTVLArray(tvl).filter((x) => x.originChainId === chainInfo.id);
  }, [chainInfo, tvl]);

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
              isAsset
            />
          </div>
        ),
      },
      {
        Header: "Quantity",
        accessor: "amount",
        disableGroupBy: true,
        Cell: (value: any) =>
          value.row?.original?.amount !== undefined
            ? numeral(value.row?.original?.amount).format("0,0.00")
            : "",
      },
      {
        Header: "Unit Price",
        accessor: "quotePrice",
        disableGroupBy: true,
        Cell: (value: any) =>
          value.row?.original?.quotePrice !== undefined
            ? numeral(value.row?.original?.quotePrice).format("0,0.00")
            : "",
      },
      {
        Header: "Value (USD)",
        id: "totalValue",
        accessor: "totalValue",
        disableGroupBy: true,
        Cell: (value: any) =>
          value.row?.original?.totalValue !== undefined
            ? formatTVL(value.row?.original?.totalValue)
            : "",
      },
    ];
  }, [
    classes.logo,
    classes.tokenContainer,
    classes.logoPositioner,
    sortTokens,
  ]);

  return (
    <MuiReactTable
      columns={tvlColumns}
      data={chainTVL || []}
      skipPageReset={false}
      initialState={{ sortBy: [{ id: "totalValue", desc: true }] }}
    />
  );
};

export default TVLTable;
