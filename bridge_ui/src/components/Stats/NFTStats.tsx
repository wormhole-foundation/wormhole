import {
  Button,
  CircularProgress,
  makeStyles,
  Typography,
} from "@material-ui/core";
import clsx from "clsx";
import numeral from "numeral";
import { useCallback, useEffect, useMemo, useState } from "react";
import useNFTTVL from "../../hooks/useNFTTVL";
import {
  BETA_CHAINS,
  CHAINS_WITH_NFT_SUPPORT,
  getNFTBridgeAddressForChain,
} from "../../utils/consts";
import NFTViewer from "../TokenSelectors/NFTViewer";
import MuiReactTable from "./tableComponents/MuiReactTable";
import {
  //DENY_LIST,
  ALLOW_LIST,
} from "./nftLists";

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
  tableBox: {
    display: "flex",
    justifyContent: "center",
    "& > *": {
      margin: theme.spacing(1),
    },
    flexWrap: "wrap",
  },
  randomButton: {
    margin: "0px auto 8px",
    display: "block",
  },
  randomNftContainer: {
    minHeight: "550px",
    maxWidth: "100%",
  },
  alignCenter: {
    margin: "0 auto",
    display: "block",
  },
  tableContainer: {
    flexGrow: 1,
    width: "fit-content",
    maxWidth: "100%",
  },
}));

const NFTStats: React.FC<any> = () => {
  const classes = useStyles();
  const nftTVL = useNFTTVL();

  //Disable this to quickly turn off
  //TODO also change what data is fetched off this
  const enableRandomNFT = true;

  const [randomNumber, setRandomNumber] = useState<number | null>(null);
  const randomNft = useMemo(
    () =>
      (randomNumber !== null && nftTVL.data && nftTVL.data[randomNumber]) ||
      null,
    [randomNumber, nftTVL.data]
  );
  const genRandomNumber = useCallback(() => {
    if (!nftTVL || !nftTVL.data || !nftTVL.data?.length || nftTVL.isFetching) {
      setRandomNumber(null);
    } else {
      let found = false;
      let nextNumber = Math.floor(Math.random() * nftTVL.data.length);

      while (!found) {
        if (!nftTVL.data) {
          return null;
        }
        const item = nftTVL?.data[nextNumber]?.mintKey?.toLowerCase() || null;
        if (ALLOW_LIST.find((x) => x.toLowerCase() === item)) {
          found = true;
        } else {
          nextNumber = Math.floor(Math.random() * nftTVL.data.length);
        }
      }

      setRandomNumber(nextNumber);
    }
  }, [nftTVL]);
  useEffect(() => {
    genRandomNumber();
  }, [nftTVL.isFetching, genRandomNumber]);

  const data = useMemo(() => {
    const output: any[] = [];
    if (nftTVL.data && !nftTVL.isFetching) {
      CHAINS_WITH_NFT_SUPPORT.filter(
        (chain) => !BETA_CHAINS.find((x) => x === chain.id)
      ).forEach((chain) => {
        output.push({
          nfts: nftTVL?.data?.filter((x) => x.chainId === chain.id),
          chainName: chain.name,
          chainId: chain.id,
          chainLogo: chain.logo,
          contractAddress: getNFTBridgeAddressForChain(chain.id),
        });
      });
    }

    return output;
  }, [nftTVL]);

  //Generate allow list
  // useEffect(() => {
  //   const output: string[] = [];
  //   if (nftTVL.data) {
  //     nftTVL.data.forEach((item) => {
  //       if (
  //         !DENY_LIST.find((x) => x.toLowerCase() === item.mintKey.toLowerCase())
  //       ) {
  //         if (!output.includes(item.mintKey)) {
  //           output.push(item.mintKey);
  //         }
  //       }
  //     });
  //   }
  //   console.log(JSON.stringify(output));
  // }, [nftTVL.data]);

  const tvlColumns = useMemo(() => {
    return [
      { Header: "Chain", accessor: "chainName", disableGroupBy: true },
      // {
      //   Header: "Address",
      //   accessor: "contractAddress",
      //   disableGroupBy: true,
      //   Cell: (value: any) =>
      //     value.row?.original?.contractAddress &&
      //     value.row?.original?.chainId ? (
      //       <SmartAddress
      //         chainId={value.row?.original?.chainId}
      //         address={value.row?.original?.contractAddress}
      //       />
      //     ) : (
      //       ""
      //     ),
      // },
      {
        Header: "NFTs Locked",
        id: "nftCount",
        accessor: "nftCount",
        align: "right",
        disableGroupBy: true,
        Cell: (value: any) =>
          value.row?.original?.nfts?.length !== undefined
            ? numeral(value.row?.original?.nfts?.length).format("0 a")
            : "",
      },
    ];
  }, []);

  const header = (
    <div className={classes.flexBox}>
      <div className={classes.explainerContainer}>
        <Typography variant="h5">Total NFTs Locked</Typography>
        <Typography variant="subtitle2" color="textSecondary">
          These NFTs are currently locked by the NFT Bridge contracts.
        </Typography>
      </div>
      <div className={classes.grower} />
      {!nftTVL.isFetching ? (
        <div
          className={clsx(classes.explainerContainer, classes.totalContainer)}
        >
          <Typography
            variant="body2"
            color="textSecondary"
            component="div"
            noWrap
          >
            {"Total "}
          </Typography>
          <Typography
            variant="h3"
            component="div"
            noWrap
            className={classes.totalValue}
          >
            {nftTVL.data?.length || "0"}
          </Typography>
        </div>
      ) : null}
    </div>
  );

  const table = (
    <MuiReactTable
      columns={tvlColumns}
      data={data || []}
      skipPageReset={false}
      initialState={{ sortBy: [{ id: "nftCount", desc: true }] }}
    />
  );

  const randomNFTContent =
    enableRandomNFT && randomNft ? (
      <div className={classes.randomNftContainer}>
        <Button
          className={classes.randomButton}
          variant="contained"
          onClick={genRandomNumber}
          color="primary"
        >
          Load Random Wormhole NFT
        </Button>
        <NFTViewer chainId={randomNft.chainId} value={randomNft} />
      </div>
    ) : null;

  // const allNfts =
  //   nftTVL?.data?.map((thing) => (
  //     <NFTViewer chainId={thing.chainId} value={thing} />
  //   )) || [];

  return (
    <>
      {header}
      {nftTVL.isFetching ? (
        <CircularProgress className={classes.alignCenter} />
      ) : (
        <div className={classes.tableBox}>
          <div className={classes.tableContainer}>{table}</div>
          {randomNFTContent}
        </div>
      )}
      {/* {allNfts} */}
    </>
  );
};

export default NFTStats;
