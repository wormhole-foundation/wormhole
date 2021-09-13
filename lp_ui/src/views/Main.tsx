import {
  Container,
  makeStyles,
  Typography,
  Paper,
  TextField,
  Button,
  Divider,
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
import parsePool from "@certusone/wormhole-sdk/lib/migration/parsePool";
import addLiquidityTx from "@certusone/wormhole-sdk/lib/migration/addLiquidity";
import claimSharesTx from "@certusone/wormhole-sdk/lib/migration/claimShares";
import migrateTokensTx from "@certusone/wormhole-sdk/lib/migration/migrateTokens";

import SolanaCreateAssociatedAddress, {
  useAssociatedAccountExistsState,
} from "../components/SolanaCreateAssociatedAddress";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";

const useStyles = makeStyles(() => ({
  rootContainer: {},
  mainPaper: {
    "& > *": {
      margin: "1rem",
    },
    padding: "2rem",
  },
  divider: {
    margin: "2rem",
  },
}));

function Main() {
  const classes = useStyles();
  const wallet = useSolanaWallet();
  const logger = useLogger();
  const connection = new Connection(SOLANA_URL, "finalized");

  const [fromMint, setFromMint] = useState("");
  const [toMint, setToMint] = useState("");
  const [shareMintAddress, setShareMintAddress] = useState<string | undefined>(
    undefined
  );

  const [poolAddress, setPoolAddress] = useState("");
  const [poolExists, setPoolExists] = useState<boolean | undefined>(undefined);
  const [poolAccountInfo, setPoolAccountInfo] = useState(undefined);
  const [parsedPoolData, setParsedPoolData] = useState(undefined);

  //These are the user's personal token accounts corresponding to the mints for the connected wallet
  const [fromTokenAccount, setFromTokenAccount] = useState<string | undefined>(
    undefined
  );
  const [toTokenAccount, setToTokenAccount] = useState<string | undefined>(
    undefined
  );
  const [shareTokenAccount, setShareTokenAccount] = useState<
    string | undefined
  >(undefined);

  //These hooks detect if the connected wallet has the requisite token accounts
  const {
    associatedAccountExists: fromTokenAccountExists,
    setAssociatedAccountExists: setFromTokenAccountExists,
  } = useAssociatedAccountExistsState(fromMint, fromTokenAccount);
  const {
    associatedAccountExists: toTokenAccountExists,
    setAssociatedAccountExists: setToTokenAccountExists,
  } = useAssociatedAccountExistsState(toMint, toTokenAccount);
  const {
    associatedAccountExists: shareTokenAccountExists,
    setAssociatedAccountExists: setShareTokenAccountExists,
  } = useAssociatedAccountExistsState(shareMintAddress, shareTokenAccount);

  //these are all the other derived information
  const [authorityAddress, setAuthorityAddress] = useState<string | undefined>(
    undefined
  );
  const [fromCustodyAddress, setFromCustodyAddress] = useState<
    string | undefined
  >(undefined);
  const [toCustodyAddress, setToCustodyAddress] = useState<string | undefined>(
    undefined
  );

  const [toggleAllData, setToggleAllData] = useState(false);

  const [liquidityAmount, setLiquidityAmount] = useState("");
  const [migrationAmount, setMigrationAmount] = useState("");
  const [redeemAmount, setRedeemAmount] = useState("");

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
      setPoolAccountInfo(undefined);
      setPoolExists(undefined);
      try {
        getMultipleAccounts(
          connection,
          [new PublicKey(poolAddress)],
          "finalized"
        ).then((result) => {
          if (result.length && result[0] !== null) {
            setPoolAccountInfo(result[0]);
            parsePool(result[0].data).then(
              (parsed) => setParsedPoolData(parsed),
              (error) => {
                logger.log("Failed to parse the pool data.");
                console.error(error);
              }
            );
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

  //Set all the addresses which derive from poolAddress
  useEffect(() => {
    getAuthorityAddress(MIGRATION_PROGRAM_ADDRESS).then((result: any) =>
      setAuthorityAddress(new PublicKey(result).toString())
    );

    getToCustodyAddress(MIGRATION_PROGRAM_ADDRESS, poolAddress).then(
      (result: any) => setToCustodyAddress(new PublicKey(result).toString())
    );
    getFromCustodyAddress(MIGRATION_PROGRAM_ADDRESS, poolAddress).then(
      (result: any) => setFromCustodyAddress(new PublicKey(result).toString())
    );
    getShareMintAddress(MIGRATION_PROGRAM_ADDRESS, poolAddress).then(
      (result: any) => setShareMintAddress(new PublicKey(result).toString())
    );
  }, [poolAddress]);

  //Set the associated token accounts when the designated mint changes
  useEffect(() => {
    if (wallet?.publicKey && fromMint) {
      Token.getAssociatedTokenAddress(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        new PublicKey(fromMint),
        wallet?.publicKey || new PublicKey([])
      ).then(
        (result) => {
          setFromTokenAccount(result.toString());
        },
        (error) => {}
      );
    }
  }, [fromMint, wallet?.publicKey]);

  useEffect(() => {
    if (wallet?.publicKey && toMint) {
      Token.getAssociatedTokenAddress(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        new PublicKey(toMint),
        wallet?.publicKey || new PublicKey([])
      ).then(
        (result) => {
          setToTokenAccount(result.toString());
        },
        (error) => {}
      );
    }
  }, [toMint, wallet?.publicKey]);

  useEffect(() => {
    if (wallet?.publicKey && shareMintAddress) {
      Token.getAssociatedTokenAddress(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        new PublicKey(shareMintAddress || ""),
        wallet?.publicKey || new PublicKey([])
      ).then(
        (result) => {
          setShareTokenAccount(result.toString());
        },
        (error) => {}
      );
    }
  }, [shareMintAddress, wallet?.publicKey]);
  /*
  End Effects!
  */

  /*
  Actions:

  These are generally onClick actions which the user can perform. They read things off the state, do something,
  and then potentially update something on the state.

  */
  const createPool = async () => {
    console.log(
      "createPool with these args",
      connection,
      wallet?.publicKey?.toString(),
      MIGRATION_PROGRAM_ADDRESS,
      fromMint,
      toMint
    );
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
          setPoolExists(undefined); //Set these to null to force a fetch on them
          setPoolAccountInfo(undefined);
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

  const addLiquidity = async () => {
    try {
      const instruction = await addLiquidityTx(
        connection,
        wallet?.publicKey?.toString() || "",
        MIGRATION_PROGRAM_ADDRESS,
        fromMint,
        toMint,
        toTokenAccount || "",
        shareTokenAccount || "",
        BigInt(liquidityAmount)
      );

      signSendAndConfirm(wallet, connection, instruction).then(
        (transaction: any) => {
          setPoolExists(undefined); //Set these to null to force a fetch on them
          setPoolAccountInfo(undefined);
          logger.log("Successfully added liquidity to the pool.");
        },
        (error) => {
          logger.log("Could not complete the addLiquidity transaction");
          console.error(error);
        }
      );
    } catch (e) {
      logger.log("Could not complete the addLiquidity transaction");
      console.error(e);
    }
  };

  const migrateTokens = async () => {
    try {
      const instruction = await migrateTokensTx(
        connection,
        wallet?.publicKey?.toString() || "",
        MIGRATION_PROGRAM_ADDRESS,
        fromMint,
        toMint,
        fromTokenAccount || "",
        toTokenAccount || "",
        BigInt(migrationAmount)
      );

      signSendAndConfirm(wallet, connection, instruction).then(
        (transaction: any) => {
          setPoolExists(undefined); //Set these to null to force a fetch on them
          setPoolAccountInfo(undefined);
          logger.log("Successfully migrated the tokens.");
        },
        (error) => {
          logger.log("Could not complete the migrateTokens transaction.");
          console.error(error);
        }
      );
    } catch (e) {
      logger.log("Could not complete the migrateTokens transaction.");
      console.error(e);
    }
  };

  const redeemShares = async () => {
    try {
      const instruction = await claimSharesTx(
        connection,
        wallet?.publicKey?.toString() || "",
        MIGRATION_PROGRAM_ADDRESS,
        fromMint,
        toMint,
        toTokenAccount || "",
        shareTokenAccount || "",
        BigInt(redeemAmount)
      );

      signSendAndConfirm(wallet, connection, instruction).then(
        (transaction: any) => {
          setPoolExists(undefined); //Set these to null to force a fetch on them
          setPoolAccountInfo(undefined);
          logger.log("Successfully redeemed the shares.");
        },
        (error) => {
          logger.log("Could not complete the claimShares transaction.");
          console.error(error);
        }
      );
    } catch (e) {
      logger.log("Could not complete the claimShares transaction.");
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
        disabled={poolExists}
      >
        {poolExists
          ? "This Pool is instantiated."
          : "This pool has not been instantiated! Click here to create it."}
      </Button>
    </div>
  );

  const addLiquidityUI = (
    <>
      <Typography variant="h4">Add Liquidity</Typography>
      <Typography variant="body1">
        This will remove 'To' tokens from your wallet, and give you an equal
        number of 'Share' tokens.
      </Typography>
      <TextField
        value={liquidityAmount}
        type="number"
        onChange={(event) => setLiquidityAmount(event.target.value)}
        label={"Amount to add"}
      ></TextField>
      <Button variant="contained" onClick={addLiquidity}>
        Add Liquidity
      </Button>
    </>
  );

  const migrateTokensUI = (
    <>
      <Typography variant="h4">Migrate Tokens</Typography>
      <Typography variant="body1">
        This will remove 'From' tokens from your wallet, and give you an equal
        number of 'To' tokens.
      </Typography>
      <TextField
        value={migrationAmount}
        type="number"
        onChange={(event) => setMigrationAmount(event.target.value)}
        label={"Amount to add"}
      ></TextField>
      <Button variant="contained" onClick={migrateTokens}>
        Migrate Tokens
      </Button>
    </>
  );

  const redeemSharesUI = (
    <>
      <Typography variant="h4">Redeem Shares</Typography>
      <Typography variant="body1">
        This will remove 'Share' tokens from your wallet, and give you an equal
        number of 'From' tokens.
      </Typography>
      <TextField
        type="number"
        value={redeemAmount}
        onChange={(event) => setRedeemAmount(event.target.value)}
        label={"Amount to add"}
      ></TextField>
      <Button variant="contained" onClick={redeemShares}>
        Redeem Shares
      </Button>
    </>
  );

  const relevantTokenAccounts = (
    <>
      <Typography variant="h4">Your Relevant Token Accounts: </Typography>
      <Typography variant="body1">
        {"'From' SPL Token Account: " + fromTokenAccount}
      </Typography>
      <SolanaCreateAssociatedAddress
        mintAddress={fromMint}
        readableTargetAddress={fromTokenAccount}
        associatedAccountExists={fromTokenAccountExists}
        setAssociatedAccountExists={setFromTokenAccountExists}
      />
      <Typography variant="body1">
        {"'To' SPL Token Account: " + toTokenAccount}
      </Typography>
      <SolanaCreateAssociatedAddress
        mintAddress={toMint}
        readableTargetAddress={toTokenAccount}
        associatedAccountExists={toTokenAccountExists}
        setAssociatedAccountExists={setToTokenAccountExists}
      />
      <Typography variant="body1">
        {"Share SPL Token Account: " + shareTokenAccount}
      </Typography>
      <SolanaCreateAssociatedAddress
        mintAddress={shareMintAddress}
        readableTargetAddress={shareTokenAccount}
        associatedAccountExists={shareTokenAccountExists}
        setAssociatedAccountExists={setShareTokenAccountExists}
      />
    </>
  );

  const poolInfo = (
    <div>
      {
        <Button
          variant="outlined"
          onClick={() => setToggleAllData(!toggleAllData)}
        >
          {toggleAllData ? "Hide Verbose Pool Data" : "Show Verbose Pool Data"}
        </Button>
      }
      {toggleAllData ? (
        <>
          <Typography>{"Pool Address: " + poolAddress}</Typography>
          <Typography>{"Pool has been instantiated: " + poolExists}</Typography>
          <Typography>{"'From' Token Mint Address: " + fromMint}</Typography>
          <Typography>{"'To' Token Mint Address: " + toMint}</Typography>
          <Typography>{"Share Token Mint: " + shareMintAddress}</Typography>
          <Typography>{"Authority Address: " + authorityAddress}</Typography>
          <Typography>
            {"'From' Custody Mint: " + fromCustodyAddress}
          </Typography>
          <Typography>{"'To' Custody Mint: " + toCustodyAddress}</Typography>
          <Typography>
            {"Full Parsed Data for Pool:  " + JSON.stringify(parsedPoolData)}
          </Typography>
        </>
      ) : null}
    </div>
  );

  const mainContent = (
    <>
      {toAndFromSelector}
      <Divider className={classes.divider} />
      {poolInfo}
      {createPoolButton}
      <Divider className={classes.divider} />
      {relevantTokenAccounts}
      <Divider className={classes.divider} />
      {addLiquidityUI}
      <Divider className={classes.divider} />
      {redeemSharesUI}
      <Divider className={classes.divider} />
      {migrateTokensUI}
    </>
  );

  const content = !wallet.publicKey ? (
    <Typography>Please connect your wallet.</Typography>
  ) : !poolAddress ? (
    toAndFromSelector
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
