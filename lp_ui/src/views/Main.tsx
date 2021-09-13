import {
  Container,
  makeStyles,
  Typography,
  Paper,
  TextField,
  Button,
} from "@material-ui/core";
//import { pool_address } from "@certusone/wormhole-sdk/lib/solana/migration/wormhole_migration";
import { useCallback, useEffect, useState } from "react";
import LogWatcher from "../components/LogWatcher";
import SolanaWalletKey from "../components/SolanaWalletKey";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import TabContext from "@material-ui/lab/TabContext";
import TabList from "@material-ui/lab/TabList";
import TabPanel from "@material-ui/lab/TabPanel";
import { MIGRATION_PROGRAM_ADDRESS, SOLANA_URL } from "../utils/consts";
import { PublicKey, Connection } from "@solana/web3.js";
import { useLogger } from "../contexts/Logger";
import { getMultipleAccounts, signSendAndConfirm } from "../utils/solana";
import getAuthorityAddress from "@certusone/wormhole-sdk/lib/migration/authorityAddress";
import createPoolAccount from "@certusone/wormhole-sdk/lib/migration/createPool";
import getPoolAddress from "@certusone/wormhole-sdk/lib/migration/poolAddress";
import getFromCustodyAddress from "@certusone/wormhole-sdk/lib/migration/fromCustodyAddress";
import getToCustodyAddress from "@certusone/wormhole-sdk/lib/migration/toCustodyAddress";
import getShareMintAddress from "@certusone/wormhole-sdk/lib/migration/shareMintAddress";

const useStyles = makeStyles(() => ({
  rootContainer: {},
  mainPaper: {
    "& > *": {
      margin: "1rem",
    },
    padding: "2rem",
  },
}));

function Main() {
  const classes = useStyles();
  const wallet = useSolanaWallet();
  const logger = useLogger();
  const connection = new Connection(SOLANA_URL, "finalized");

  const [fromMint, setFromMint] = useState("");
  const [toMint, setToMint] = useState("");

  const [poolAddress, setPoolAddress] = useState("");
  const [poolExists, setPoolExists] = useState<boolean | null>(null);
  const [poolAccountInfo, setPoolAccountInfo] = useState(null);
  const [shareTokenMint, setShareTokenMint] = useState(null);
  const [toTokenAccount, setToTokenAccount] = useState(null);
  const [fromTokenAccount, setFromTokenAccount] = useState(null);
  const [shareTokenAccount, setShareTokenAccount] = useState(null);

  //these are all the other derived information
  const [authorityAddress, setAuthorityAddress] = useState(null);
  const [fromCustodyAddress, setFromCustodyAddress] = useState(null);
  const [toCustodyAddress, setToCustodyAddress] = useState(null);
  const [shareMintAddress, setShareMintAddress] = useState(null);

  /*
  Effects***

  These are generally data fetchers which fire when requisite data populates.

  */
  //Retrieve pool address on selectedTokens change
  useEffect(() => {
    if (toMint && fromMint) {
      setPoolAddress("");
      getPoolAddress(MIGRATION_PROGRAM_ADDRESS, fromMint, toMint).then(
        (result) => {
          const key = new PublicKey(result).toString();
          logger.log("Calculated the pool address at: " + key);
          setPoolAddress(key);
        },
        (error) => logger.log("ERROR, could not calculate pool address.")
      );
    }
  }, [toMint, fromMint, setPoolAddress]);

  //Retrieve the poolAccount every time the pool address changes.
  useEffect(() => {
    if (poolAddress) {
      setPoolAccountInfo(null);
      setPoolExists(null);
      try {
        getMultipleAccounts(
          connection,
          [new PublicKey(poolAddress)],
          "finalized"
        ).then((result) => {
          if (result.length && result[0] !== null) {
            setPoolAccountInfo(result[0]);
            setPoolExists(true);
            logger.log("Successfully found account info for the pool.");
          } else if (result.length && result[0] === null) {
            logger.log("Confirmed that the pool does not exist.");
            setPoolExists(false);
          } else {
            logger.log(
              "unexpected error in fetching pool address. Please reload and try again"
            );
          }
        });
      } catch (e) {
        logger.log("Could not fetch pool address");
      }
    }
  }, [poolAddress]);

  useEffect(() => {
    getAuthorityAddress(MIGRATION_PROGRAM_ADDRESS).then((result: any) =>
      setAuthorityAddress(result)
    );

    getToCustodyAddress(MIGRATION_PROGRAM_ADDRESS, poolAddress).then(
      (result: any) => setToCustodyAddress(result)
    );
    getFromCustodyAddress(MIGRATION_PROGRAM_ADDRESS, poolAddress).then(
      (result: any) => setFromCustodyAddress(result)
    );
    getShareMintAddress(MIGRATION_PROGRAM_ADDRESS, poolAddress).then(
      (result: any) => setShareMintAddress(result)
    );
  }, [poolAddress]);
  /*
  End Effects!
  */

  /*
  Actions:

  These are generally onClick actions which the user can perform. They read things off the state, do something,
  and then potentially update something on the state.

  */
  const createPool = async () => {
    try {
      const instruction = await createPoolAccount(
        connection,
        wallet?.publicKey?.toString() || "",
        MIGRATION_PROGRAM_ADDRESS,
        wallet?.publicKey?.toString() || "",
        fromMint,
        toMint
      );

      signSendAndConfirm(wallet, connection, instruction).then(
        (transaction: any) => {
          setPoolExists(null); //Set these to null to force a fetch on them
          setPoolAccountInfo(null);
          logger.log("Successfully created the pool.");
        },
        (error) => {
          logger.log("Could not create the pool");
          console.error(error);
        }
      );
    } catch (e) {
      logger.log("Failed to create the pool.");
      console.error(e);
    }
  };
  /*
  End actions!
  */

  const toAndFromSelector = (
    <>
      <Typography>
        Please enter the mint addresses for the 'To' and 'From' tokens you're
        interested in.
      </Typography>
      <TextField
        value={fromMint}
        onChange={(event) => setFromMint(event.target.value)}
        label={"From Token"}
        fullWidth
        style={{ display: "block" }}
      ></TextField>
      <TextField
        value={toMint}
        onChange={(event) => setToMint(event.target.value)}
        label={"To Token"}
        fullWidth
        style={{ display: "block" }}
      ></TextField>
    </>
  );

  const createPoolButton = (
    <div>
      <Button
        variant="contained"
        onClick={() => createPool()}
        disabled={poolExists || !poolAddress}
      >
        Click here to instantiate the pool for these tokens.
      </Button>
    </div>
  );

  const addLiquidity = (
    <>
      <Typography>
        Add 'to' tokens to this pool, and receive liquidity tokens.
      </Typography>
      <TextField
        value={toMint}
        onChange={(event) => setToMint(event.target.value)}
        label={"To Token"}
      ></TextField>
    </>
  );

  const mainContent = (
    <>
      {toAndFromSelector}
      {createPoolButton}
    </>
  );

  const content = !wallet.publicKey ? (
    <Typography>Please connect your wallet.</Typography>
  ) : (
    mainContent
  );

  return (
    <Container maxWidth="md" className={classes.rootContainer}>
      <Paper className={classes.mainPaper}>
        <SolanaWalletKey />
        {content}
      </Paper>
      <LogWatcher />
    </Container>
  );
}

export default Main;
