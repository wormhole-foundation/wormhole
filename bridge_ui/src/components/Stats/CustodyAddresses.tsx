import {
  CHAIN_ID_AVAX,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_POLYGON,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { makeStyles, Typography } from "@material-ui/core";
import { useMemo } from "react";
import {
  getNFTBridgeAddressForChain,
  getTokenBridgeAddressForChain,
  SOL_CUSTODY_ADDRESS,
  SOL_NFT_CUSTODY_ADDRESS,
} from "../../utils/consts";
import SmartAddress from "../SmartAddress";
import MuiReactTable from "./tableComponents/MuiReactTable";

const useStyles = makeStyles((theme) => ({
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
}));

const CustodyAddresses: React.FC<any> = () => {
  const classes = useStyles();
  const data = useMemo(() => {
    return [
      {
        chainName: "Ethereum",
        chainId: CHAIN_ID_ETH,
        tokenAddress: getTokenBridgeAddressForChain(CHAIN_ID_ETH),
        nftAddress: getNFTBridgeAddressForChain(CHAIN_ID_ETH),
      },
      {
        chainName: "Solana",
        chainId: CHAIN_ID_SOLANA,
        tokenAddress: SOL_CUSTODY_ADDRESS,
        nftAddress: SOL_NFT_CUSTODY_ADDRESS,
      },
      {
        chainName: "Binance Smart Chain",
        chainId: CHAIN_ID_BSC,
        tokenAddress: getTokenBridgeAddressForChain(CHAIN_ID_BSC),
        nftAddress: getNFTBridgeAddressForChain(CHAIN_ID_BSC),
      },
      {
        chainName: "Terra",
        chainId: CHAIN_ID_TERRA,
        tokenAddress: getTokenBridgeAddressForChain(CHAIN_ID_TERRA),
        nftAddress: null,
      },
      {
        chainName: "Polygon",
        chainId: CHAIN_ID_POLYGON,
        tokenAddress: getTokenBridgeAddressForChain(CHAIN_ID_POLYGON),
        nftAddress: getNFTBridgeAddressForChain(CHAIN_ID_POLYGON),
      },
      {
        chainName: "Avalanche",
        chainId: CHAIN_ID_AVAX,
        tokenAddress: getTokenBridgeAddressForChain(CHAIN_ID_AVAX),
        nftAddress: getNFTBridgeAddressForChain(CHAIN_ID_AVAX),
      },
    ];
  }, []);

  const tvlColumns = useMemo(() => {
    return [
      { Header: "Chain", accessor: "chainName", disableGroupBy: true },
      {
        Header: "Token Address",
        id: "tokenAddress",
        accessor: "address",
        disableGroupBy: true,
        Cell: (value: any) =>
          value.row?.original?.tokenAddress && value.row?.original?.chainId ? (
            <SmartAddress
              chainId={value.row?.original?.chainId}
              address={value.row?.original?.tokenAddress}
            />
          ) : (
            ""
          ),
      },
      {
        Header: "NFT Address",
        id: "nftAddress",
        accessor: "address",
        disableGroupBy: true,
        Cell: (value: any) =>
          value.row?.original?.nftAddress && value.row?.original?.chainId ? (
            <SmartAddress
              chainId={value.row?.original?.chainId}
              address={value.row?.original?.nftAddress}
            />
          ) : (
            ""
          ),
      },
    ];
  }, []);

  const header = (
    <div className={classes.flexBox}>
      <div className={classes.explainerContainer}>
        <Typography variant="h5">Custody Addresses</Typography>
        <Typography variant="subtitle2" color="textSecondary">
          These are the custody addresses which hold collateralized assets for
          the token bridge.
        </Typography>
      </div>
      <div className={classes.grower} />
    </div>
  );

  const table = (
    <MuiReactTable
      columns={tvlColumns}
      data={data || []}
      skipPageReset={false}
      initialState={{}}
    />
  );

  return (
    <>
      {header}
      {table}
    </>
  );
};

export default CustodyAddresses;
