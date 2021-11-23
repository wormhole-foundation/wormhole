import addLiquidityTx from "@certusone/wormhole-sdk/lib/esm/migration/addLiquidity";
import getAuthorityAddress from "@certusone/wormhole-sdk/lib/esm/migration/authorityAddress";
import claimSharesTx from "@certusone/wormhole-sdk/lib/esm/migration/claimShares";
import createPoolAccount from "@certusone/wormhole-sdk/lib/esm/migration/createPool";
import getFromCustodyAddress from "@certusone/wormhole-sdk/lib/esm/migration/fromCustodyAddress";
import migrateTokensTx from "@certusone/wormhole-sdk/lib/esm/migration/migrateTokens";
import parsePool from "@certusone/wormhole-sdk/lib/esm/migration/parsePool";
import getPoolAddress from "@certusone/wormhole-sdk/lib/esm/migration/poolAddress";
import removeLiquidityTx from "@certusone/wormhole-sdk/lib/esm/migration/removeLiquidity";
import getShareMintAddress from "@certusone/wormhole-sdk/lib/esm/migration/shareMintAddress";
import getToCustodyAddress from "@certusone/wormhole-sdk/lib/esm/migration/toCustodyAddress";
import {
  Button,
  CircularProgress,
  Container,
  Divider,
  makeStyles,
  Paper,
  TextField,
  Typography,
} from "@material-ui/core";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, PublicKey } from "@solana/web3.js";
//import { pool_address } from "@certusone/wormhole-sdk/lib/esm/solana/migration/wormhole_migration";
import { parseUnits } from "ethers/lib/utils";
import { useCallback, useEffect, useMemo, useState } from "react";
import LogWatcher from "../components/LogWatcher";
import SolanaCreateAssociatedAddress, {
  useAssociatedAccountExistsState,
} from "../components/SolanaCreateAssociatedAddress";
import SolanaWalletKey from "../components/SolanaWalletKey";
import { useLogger } from "../contexts/Logger";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import { MIGRATION_PROGRAM_ADDRESS, SOLANA_URL } from "../utils/consts";
import { getMultipleAccounts, signSendAndConfirm } from "../utils/solana";

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
  spacer: {
    height: "1rem",
  },
}));

export const compareWithDecimalOffset = (
  valueA: string,
  decimalsA: number,
  valueB: string,
  decimalsB: number
) => {
  //find which is larger, and offset by that amount
  const decimalsBasis = decimalsA > decimalsB ? decimalsA : decimalsB;
  const normalizedA = parseUnits(valueA, decimalsBasis).toBigInt();
  const normalizedB = parseUnits(valueB, decimalsBasis).toBigInt();

  if (normalizedA < normalizedB) {
    return -1;
  } else if (normalizedA === normalizedB) {
    return 0;
  } else {
    return 1;
  }
};

const getDecimals = async (
  connection: Connection,
  mint: string,
  setter: (decimals: number | undefined) => void,
  log: (value: string, type?: "error" | "info" | "success" | undefined) => void
) => {
  setter(undefined);
  if (mint) {
    try {
      const pk = new PublicKey(mint);
      const info = await connection.getParsedAccountInfo(pk);
      // @ts-ignore
      const decimals = info.value?.data.parsed.info.decimals;
      log(`${mint} has ${decimals} decimals`);
      setter(decimals);
    } catch (e) {
      log(`Unable to determine decimals of ${mint}`);
    }
  }
};

const getBalance = async (
  connection: Connection,
  address: string | undefined,
  setter: (balance: string | undefined) => void,
  log: (value: string, type?: "error" | "info" | "success" | undefined) => void
) => {
  setter(undefined);
  if (address) {
    try {
      const pk = new PublicKey(address);
      const info = await connection.getParsedAccountInfo(pk);
      // @ts-ignore
      const balance = info.value?.data.parsed.info.tokenAmount.uiAmountString;
      log(`${address} has a balance of ${balance}`);
      setter(balance);
    } catch (e) {
      log(`Unable to determine balance of ${address}`, "error");
    }
  }
};

function Main() {
  const classes = useStyles();
  const wallet = useSolanaWallet();
  const { log } = useLogger();
  const connection = useMemo(() => new Connection(SOLANA_URL, "confirmed"), []);

  const [fromMintHolder, setFromMintHolder] = useState("");
  const [fromMintDecimals, setFromMintDecimals] = useState<number | undefined>(
    undefined
  );
  const [toMintHolder, setToMintHolder] = useState("");
  const [toMintDecimals, setToMintDecimals] = useState<number | undefined>(
    undefined
  );
  const [shareMintAddress, setShareMintAddress] = useState<string | undefined>(
    undefined
  );
  const [shareMintDecimals, setShareMintDecimals] = useState<any>(undefined);

  let fromMint: string = "";
  let toMint: string = "";
  try {
    fromMint = fromMintHolder && new PublicKey(fromMintHolder).toString();
    toMint = toMintHolder && new PublicKey(toMintHolder).toString();
  } catch (e) {}

  const [poolAddress, setPoolAddress] = useState("");
  const [poolExists, setPoolExists] = useState<boolean | undefined>(undefined);
  const [poolAccountInfo, setPoolAccountInfo] = useState<any>(undefined);
  const [parsedPoolData, setParsedPoolData] = useState(undefined);

  //These are the user's personal token accounts corresponding to the mints for the connected wallet
  const [fromTokenAccount, setFromTokenAccount] = useState<string | undefined>(
    undefined
  );
  const [fromTokenAccountBalance, setFromTokenAccountBalance] = useState<
    string | undefined
  >();
  const [toTokenAccount, setToTokenAccount] = useState<string | undefined>(
    undefined
  );
  const [toTokenAccountBalance, setToTokenAccountBalance] = useState<
    string | undefined
  >();
  const [shareTokenAccount, setShareTokenAccount] = useState<
    string | undefined
  >(undefined);
  const [shareTokenAccountBalance, setShareTokenAccountBalance] = useState<
    string | undefined
  >();

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
  const [fromCustodyBalance, setFromCustodyBalance] = useState<
    string | undefined
  >(undefined);
  const [toCustodyAddress, setToCustodyAddress] = useState<string | undefined>(
    undefined
  );
  const [toCustodyBalance, setToCustodyBalance] = useState<string | undefined>(
    undefined
  );

  const [toggleAllData, setToggleAllData] = useState(false);

  const [liquidityAmount, setLiquidityAmount] = useState("");
  const [removeLiquidityAmount, setRemoveLiquidityAmount] = useState("");
  const [migrationAmount, setMigrationAmount] = useState("");
  const [redeemAmount, setRedeemAmount] = useState("");

  const [liquidityIsProcessing, setLiquidityIsProcessing] = useState(false);
  const [removeLiquidityIsProcessing, setRemoveLiquidityIsProcessing] =
    useState(false);
  const [migrationIsProcessing, setMigrationIsProcessing] = useState(false);
  const [redeemIsProcessing, setRedeemIsProcessing] = useState(false);
  const [createPoolIsProcessing, setCreatePoolIsProcessing] = useState(false);

  /*
  Effects***

  These are generally data fetchers which fire when requisite data populates.

  */
  //Retrieve from mint information when fromMint changes
  useEffect(() => {
    getDecimals(connection, fromMint, setFromMintDecimals, log);
  }, [connection, fromMint, log]);

  //Retrieve to mint information when fromMint changes
  useEffect(() => {
    getDecimals(connection, toMint, setToMintDecimals, log);
  }, [connection, toMint, log]);

  //Retrieve to mint information when shareMint changes
  useEffect(() => {
    // TODO: cancellable
    if (shareMintAddress) {
      getDecimals(connection, shareMintAddress, setShareMintDecimals, log);
    } else {
      setShareMintDecimals(undefined);
    }
  }, [connection, shareMintAddress, log]);

  //Retrieve from custody balance when fromCustodyAccount changes
  useEffect(() => {
    // TODO: cancellable
    if (fromCustodyAddress) {
      getBalance(connection, fromCustodyAddress, setFromCustodyBalance, log);
    } else {
      setFromCustodyBalance(undefined);
    }
  }, [connection, fromCustodyAddress, log]);

  //Retrieve from custody balance when toCustodyAccount changes
  useEffect(() => {
    // TODO: cancellable
    if (toCustodyAddress) {
      getBalance(connection, toCustodyAddress, setToCustodyBalance, log);
    } else {
      setFromCustodyBalance(undefined);
    }
  }, [connection, toCustodyAddress, log]);

  useEffect(() => {
    if (fromTokenAccountExists) {
      getBalance(connection, fromTokenAccount, setFromTokenAccountBalance, log);
    }
  }, [connection, fromTokenAccount, fromTokenAccountExists, log]);
  useEffect(() => {
    if (toTokenAccountExists) {
      getBalance(connection, toTokenAccount, setToTokenAccountBalance, log);
    }
  }, [connection, toTokenAccount, toTokenAccountExists, log]);
  useEffect(() => {
    if (shareTokenAccountExists) {
      getBalance(
        connection,
        shareTokenAccount,
        setShareTokenAccountBalance,
        log
      );
    }
  }, [connection, shareTokenAccount, shareTokenAccountExists, log]);

  //Retrieve pool address on selectedTokens change
  useEffect(() => {
    if (toMint && fromMint) {
      setPoolAddress("");
      setPoolExists(undefined);
      setPoolAccountInfo(undefined);
      setParsedPoolData(undefined);
      getPoolAddress(MIGRATION_PROGRAM_ADDRESS, fromMint, toMint).then(
        (result) => {
          const key = new PublicKey(result).toString();
          log("Calculated the pool address at: " + key);
          setPoolAddress(key);
        },
        (error) => log("Could not calculate pool address.", "error")
      );
    }
  }, [log, toMint, fromMint, setPoolAddress]);

  //Retrieve the poolAccount every time the pool address changes.
  useEffect(() => {
    console.log(
      "fired the poolAccountInfo effect",
      poolAddress,
      poolAccountInfo
    );
    if (poolAddress && poolAccountInfo === undefined) {
      setPoolExists(undefined);
      try {
        getMultipleAccounts(
          connection,
          [new PublicKey(poolAddress)],
          "confirmed"
        ).then((result) => {
          if (result.length && result[0] !== null) {
            setPoolAccountInfo(result[0]);
            parsePool(result[0].data).then(
              (parsed) => setParsedPoolData(parsed),
              (error) => {
                log("Failed to parse the pool data.", "error");
                console.error(error);
              }
            );
            setPoolExists(true);
            log("Successfully found account info for the pool.");
          } else if (result.length && result[0] === null) {
            log("Confirmed that the pool does not exist.");
            setPoolExists(false);
            setPoolAccountInfo(null);
          } else {
            log(
              "unexpected error in fetching pool address. Please reload and try again",
              "error"
            );
          }
        });
      } catch (e) {
        log("Could not fetch pool address", "error");
      }
    }
  }, [connection, log, poolAddress, poolAccountInfo]);

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
  const refreshPoolBalances = useCallback(() => {
    getBalance(connection, fromCustodyAddress, setFromCustodyBalance, log);
    getBalance(connection, toCustodyAddress, setToCustodyBalance, log);
  }, [connection, fromCustodyAddress, toCustodyAddress, log]);

  const refreshWalletBalances = useCallback(() => {
    if (fromTokenAccountExists) {
      getBalance(connection, fromTokenAccount, setFromTokenAccountBalance, log);
    }
    if (toTokenAccountExists) {
      getBalance(connection, toTokenAccount, setToTokenAccountBalance, log);
    }
    if (shareTokenAccountExists) {
      getBalance(
        connection,
        shareTokenAccount,
        setShareTokenAccountBalance,
        log
      );
    }
  }, [
    connection,
    fromTokenAccount,
    toTokenAccount,
    shareTokenAccount,
    fromTokenAccountExists,
    toTokenAccountExists,
    shareTokenAccountExists,
    log,
  ]);

  const createPool = useCallback(async () => {
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
      setCreatePoolIsProcessing(true);
      signSendAndConfirm(wallet, connection, instruction).then(
        (transaction: any) => {
          setPoolExists(undefined); //Set these to null to force a fetch on them
          setPoolAccountInfo(undefined);
          log("Successfully created the pool.", "success");
          setCreatePoolIsProcessing(false);
        },
        (error) => {
          log("Could not create the pool", "error");
          console.error(error);
          setCreatePoolIsProcessing(false);
        }
      );
    } catch (e) {
      log("Failed to create the pool.", "error");
      console.error(e);
      setCreatePoolIsProcessing(false);
    }
  }, [connection, fromMint, toMint, wallet, log]);

  const addLiquidity = useCallback(async () => {
    try {
      const instruction = await addLiquidityTx(
        connection,
        wallet?.publicKey?.toString() || "",
        MIGRATION_PROGRAM_ADDRESS,
        fromMint,
        toMint,
        toTokenAccount || "",
        shareTokenAccount || "",
        parseUnits(liquidityAmount, toMintDecimals).toBigInt()
      );
      setLiquidityIsProcessing(true);
      signSendAndConfirm(wallet, connection, instruction).then(
        (transaction: any) => {
          log("Successfully added liquidity to the pool.", "success");
          getBalance(
            connection,
            fromCustodyAddress,
            setFromCustodyBalance,
            log
          );
          getBalance(connection, toCustodyAddress, setToCustodyBalance, log);
          refreshWalletBalances();
          setLiquidityIsProcessing(false);
        },
        (error) => {
          log("Could not complete the addLiquidity transaction", "error");
          console.error(error);
          setLiquidityIsProcessing(false);
        }
      );
    } catch (e) {
      log("Could not complete the addLiquidity transaction", "error");
      console.error(e);
      setLiquidityIsProcessing(false);
    }
  }, [
    connection,
    fromMint,
    liquidityAmount,
    shareTokenAccount,
    toMint,
    toTokenAccount,
    wallet,
    log,
    toMintDecimals,
    fromCustodyAddress,
    toCustodyAddress,
    refreshWalletBalances,
  ]);

  const removeLiquidity = useCallback(async () => {
    try {
      const instruction = await removeLiquidityTx(
        connection,
        wallet?.publicKey?.toString() || "",
        MIGRATION_PROGRAM_ADDRESS,
        fromMint,
        toMint,
        toTokenAccount || "",
        shareTokenAccount || "",
        parseUnits(removeLiquidityAmount, shareMintDecimals).toBigInt()
      );
      setRemoveLiquidityIsProcessing(true);
      signSendAndConfirm(wallet, connection, instruction).then(
        (transaction: any) => {
          log("Successfully removed liquidity to the pool.", "success");
          getBalance(
            connection,
            fromCustodyAddress,
            setFromCustodyBalance,
            log
          );
          getBalance(connection, toCustodyAddress, setToCustodyBalance, log);
          refreshWalletBalances();
          setRemoveLiquidityIsProcessing(false);
        },
        (error) => {
          log("Could not complete the removeLiquidity transaction", "error");
          console.error(error);
          setRemoveLiquidityIsProcessing(false);
        }
      );
    } catch (e) {
      log("Could not complete the removeLiquidity transaction", "error");
      console.error(e);
      setRemoveLiquidityIsProcessing(false);
    }
  }, [
    connection,
    fromMint,
    removeLiquidityAmount,
    shareTokenAccount,
    toMint,
    toTokenAccount,
    wallet,
    log,
    shareMintDecimals,
    fromCustodyAddress,
    toCustodyAddress,
    refreshWalletBalances,
  ]);

  const migrateTokens = useCallback(async () => {
    try {
      const instruction = await migrateTokensTx(
        connection,
        wallet?.publicKey?.toString() || "",
        MIGRATION_PROGRAM_ADDRESS,
        fromMint,
        toMint,
        fromTokenAccount || "",
        toTokenAccount || "",
        parseUnits(migrationAmount, fromMintDecimals).toBigInt()
      );
      setMigrationIsProcessing(true);
      signSendAndConfirm(wallet, connection, instruction).then(
        (transaction: any) => {
          log("Successfully migrated the tokens.", "success");
          getBalance(
            connection,
            fromCustodyAddress,
            setFromCustodyBalance,
            log
          );
          getBalance(connection, toCustodyAddress, setToCustodyBalance, log);
          refreshWalletBalances();
          setMigrationIsProcessing(false);
        },
        (error) => {
          log("Could not complete the migrateTokens transaction.", "error");
          console.error(error);
          setMigrationIsProcessing(false);
        }
      );
    } catch (e) {
      log("Could not complete the migrateTokens transaction.", "error");
      console.error(e);
      setMigrationIsProcessing(false);
    }
  }, [
    connection,
    fromMint,
    fromTokenAccount,
    log,
    migrationAmount,
    toMint,
    toTokenAccount,
    wallet,
    fromMintDecimals,
    fromCustodyAddress,
    toCustodyAddress,
    refreshWalletBalances,
  ]);

  const redeemShares = useCallback(async () => {
    try {
      const instruction = await claimSharesTx(
        connection,
        wallet?.publicKey?.toString() || "",
        MIGRATION_PROGRAM_ADDRESS,
        fromMint,
        toMint,
        fromTokenAccount || "",
        shareTokenAccount || "",
        parseUnits(redeemAmount, shareMintDecimals).toBigInt()
      );
      setRedeemIsProcessing(true);
      signSendAndConfirm(wallet, connection, instruction).then(
        (transaction: any) => {
          log("Successfully redeemed the shares.", "success");
          getBalance(
            connection,
            fromCustodyAddress,
            setFromCustodyBalance,
            log
          );
          getBalance(connection, toCustodyAddress, setToCustodyBalance, log);
          refreshWalletBalances();
          setRedeemIsProcessing(false);
        },
        (error) => {
          log("Could not complete the claimShares transaction.", "error");
          console.error(error);
          setRedeemIsProcessing(false);
        }
      );
    } catch (e) {
      log("Could not complete the claimShares transaction.", "error");
      console.error(e);
      setRedeemIsProcessing(false);
    }
  }, [
    connection,
    fromMint,
    log,
    redeemAmount,
    shareTokenAccount,
    toMint,
    fromTokenAccount,
    wallet,
    shareMintDecimals,
    fromCustodyAddress,
    toCustodyAddress,
    refreshWalletBalances,
  ]);
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
        value={fromMintHolder}
        onChange={(event) => setFromMintHolder(event.target.value)}
        label={"From Token"}
        fullWidth
        style={{ display: "block" }}
      ></TextField>
      <TextField
        value={toMintHolder}
        onChange={(event) => setToMintHolder(event.target.value)}
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
        disabled={poolExists || createPoolIsProcessing}
      >
        {poolExists
          ? "This Pool is instantiated."
          : "This pool has not been instantiated! Click here to create it."}
      </Button>
      {createPoolIsProcessing ? <CircularProgress /> : null}
    </div>
  );

  const addLiquidityIsReady =
    poolExists &&
    shareTokenAccountExists &&
    toTokenAccountBalance &&
    liquidityAmount &&
    toMintDecimals &&
    compareWithDecimalOffset(
      liquidityAmount,
      toMintDecimals,
      toTokenAccountBalance,
      toMintDecimals
    ) !== 1;
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
      <Button
        variant="contained"
        onClick={addLiquidity}
        disabled={liquidityIsProcessing || !addLiquidityIsReady}
      >
        Add Liquidity
      </Button>
      {liquidityIsProcessing ? <CircularProgress /> : null}
    </>
  );

  const removeLiquidityIsReady =
    poolExists &&
    shareTokenAccountBalance &&
    toCustodyBalance &&
    removeLiquidityAmount &&
    toMintDecimals &&
    shareMintDecimals &&
    compareWithDecimalOffset(
      removeLiquidityAmount,
      shareMintDecimals,
      toCustodyBalance,
      toMintDecimals
    ) !== 1;
  const removeLiquidityUI = (
    <>
      <Typography variant="h4">Remove Liquidity</Typography>
      <Typography variant="body1">
        This will remove 'Share' tokens from your wallet, and give you an equal
        number of 'To' tokens.
      </Typography>
      <TextField
        value={removeLiquidityAmount}
        type="number"
        onChange={(event) => setRemoveLiquidityAmount(event.target.value)}
        label={"Amount to remove"}
      ></TextField>
      <Button
        variant="contained"
        onClick={removeLiquidity}
        disabled={removeLiquidityIsProcessing || !removeLiquidityIsReady}
      >
        Remove Liquidity
      </Button>
      {removeLiquidityIsProcessing ? <CircularProgress /> : null}
    </>
  );

  const migrateIsReady =
    poolExists &&
    fromTokenAccountBalance &&
    toCustodyBalance &&
    migrationAmount &&
    toMintDecimals &&
    fromMintDecimals &&
    compareWithDecimalOffset(
      migrationAmount,
      fromMintDecimals,
      toCustodyBalance,
      toMintDecimals
    ) !== 1;
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
        label={"Amount to migrate"}
      ></TextField>
      <Button
        variant="contained"
        onClick={migrateTokens}
        disabled={migrationIsProcessing || !migrateIsReady}
      >
        Migrate Tokens
      </Button>
      {migrationIsProcessing ? <CircularProgress /> : null}
    </>
  );

  const redeemIsReady =
    poolExists &&
    fromCustodyBalance &&
    shareTokenAccountBalance &&
    redeemAmount &&
    shareMintDecimals &&
    fromMintDecimals &&
    compareWithDecimalOffset(
      redeemAmount,
      shareMintDecimals,
      fromCustodyBalance,
      fromMintDecimals
    ) !== 1;
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
        label={"Amount to redeem"}
      ></TextField>
      <Button
        variant="contained"
        onClick={redeemShares}
        disabled={redeemIsProcessing || !redeemIsReady}
      >
        Redeem Shares
      </Button>
      {redeemIsProcessing ? <CircularProgress /> : null}
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
      {fromTokenAccountExists ? (
        <Typography>Balance: {fromTokenAccountBalance}</Typography>
      ) : null}
      <div className={classes.spacer} />
      <Typography variant="body1">
        {"'To' SPL Token Account: " + toTokenAccount}
      </Typography>
      <SolanaCreateAssociatedAddress
        mintAddress={toMint}
        readableTargetAddress={toTokenAccount}
        associatedAccountExists={toTokenAccountExists}
        setAssociatedAccountExists={setToTokenAccountExists}
      />
      {toTokenAccountExists ? (
        <Typography>Balance: {toTokenAccountBalance}</Typography>
      ) : null}
      <div className={classes.spacer} />
      <Typography variant="body1">
        {"Share SPL Token Account: " + shareTokenAccount}
      </Typography>
      <SolanaCreateAssociatedAddress
        mintAddress={shareMintAddress}
        readableTargetAddress={shareTokenAccount}
        associatedAccountExists={shareTokenAccountExists}
        setAssociatedAccountExists={setShareTokenAccountExists}
      />
      {shareTokenAccountExists ? (
        <Typography>Balance: {shareTokenAccountBalance}</Typography>
      ) : null}
      <div className={classes.spacer} />
      <Button variant="outlined" onClick={refreshWalletBalances}>
        Refresh Account Balances
      </Button>
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
      <Typography>'From' Balance In Pool</Typography>
      <Typography>{fromCustodyBalance}</Typography>
      <Typography>'To' Balance In Pool</Typography>
      <Typography>{toCustodyBalance}</Typography>
      <Button variant="outlined" onClick={refreshPoolBalances}>
        Reload Balances
      </Button>
      <Divider className={classes.divider} />
      {addLiquidityUI}
      <Divider className={classes.divider} />
      {removeLiquidityUI}
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
    <>
      <Container maxWidth="md" className={classes.rootContainer}>
        <Paper className={classes.mainPaper}>
          <SolanaWalletKey />
          {content}
        </Paper>
        <LogWatcher />
      </Container>
    </>
  );
}

export default Main;
