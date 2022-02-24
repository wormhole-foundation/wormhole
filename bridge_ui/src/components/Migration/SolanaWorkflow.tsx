import { CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import migrateTokensTx from "@certusone/wormhole-sdk/lib/esm/migration/migrateTokens";
import getPoolAddress from "@certusone/wormhole-sdk/lib/esm/migration/poolAddress";
import getToCustodyAddress from "@certusone/wormhole-sdk/lib/esm/migration/toCustodyAddress";
import { makeStyles, Typography } from "@material-ui/core";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, PublicKey } from "@solana/web3.js";
import { parseUnits } from "ethers/lib/utils";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSolanaWallet } from "../../contexts/SolanaWalletContext";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import useMetaplexData from "../../hooks/useMetaplexData";
import useSolanaTokenMap from "../../hooks/useSolanaTokenMap";
import { COLORS } from "../../muiTheme";
import { MIGRATION_PROGRAM_ADDRESS, SOLANA_HOST } from "../../utils/consts";
import { getMultipleAccounts, signSendAndConfirm } from "../../utils/solana";
import ButtonWithLoader from "../ButtonWithLoader";
import NumberTextField from "../NumberTextField";
import ShowTx from "../ShowTx";
import SmartAddress from "../SmartAddress";
import SolanaCreateAssociatedAddress, {
  useAssociatedAccountExistsState,
} from "../SolanaCreateAssociatedAddress";
import SolanaWalletKey from "../SolanaWalletKey";

const useStyles = makeStyles(() => ({
  mainPaper: {
    backgroundColor: COLORS.whiteWithTransparency,
    textAlign: "center",
    padding: "2rem",
    "& > h, p ": {
      margin: ".5rem",
    },
  },
  divider: {
    margin: "2rem 0rem 2rem 0rem",
  },
  spacer: {
    height: "2rem",
  },
}));

//TODO move to utils/solana
const getDecimals = async (
  connection: Connection,
  mint: string,
  setter: (decimals: number | undefined) => void
) => {
  setter(undefined);
  if (mint) {
    try {
      const pk = new PublicKey(mint);
      const info = await connection.getParsedAccountInfo(pk);
      // @ts-ignore
      const decimals = info.value?.data.parsed.info.decimals;
      setter(decimals);
    } catch (e) {
      console.log(`Unable to determine decimals of ${mint}`);
    }
  }
};

//TODO move to utils/solana
const getBalance = async (
  connection: Connection,
  address: string | undefined,
  setter: (balance: string | undefined) => void
) => {
  setter(undefined);
  if (address) {
    try {
      const pk = new PublicKey(address);
      const info = await connection.getParsedAccountInfo(pk);
      // @ts-ignore
      const balance = info.value?.data.parsed.info.tokenAmount.uiAmountString;
      setter(balance);
    } catch (e) {
      console.log(`Unable to determine balance of ${address}`);
    }
  }
};

export default function Workflow({
  fromMint,
  toMint,
  fromTokenAccount,
}: {
  fromMint: string;
  toMint: string;
  fromTokenAccount: string;
}) {
  const classes = useStyles();

  const connection = useMemo(
    () => new Connection(SOLANA_HOST, "confirmed"),
    []
  );
  const wallet = useSolanaWallet();
  const { isReady } = useIsWalletReady(CHAIN_ID_SOLANA);
  const solanaTokenMap = useSolanaTokenMap();
  const metaplexArray = useMemo(() => [fromMint, toMint], [fromMint, toMint]);
  const metaplexData = useMetaplexData(metaplexArray);

  const [poolAddress, setPoolAddress] = useState("");
  const [poolExists, setPoolExists] = useState<boolean | undefined>(undefined);
  const [fromTokenAccountBalance, setFromTokenAccountBalance] = useState<
    string | undefined
  >(undefined);
  const [toTokenAccount, setToTokenAccount] = useState<string | undefined>(
    undefined
  );
  const [toTokenAccountBalance, setToTokenAccountBalance] = useState<
    string | undefined
  >(undefined);
  const [fromMintDecimals, setFromMintDecimals] = useState<number | undefined>(
    undefined
  );

  const {
    associatedAccountExists: fromTokenAccountExists,
    //setAssociatedAccountExists: setFromTokenAccountExists,
  } = useAssociatedAccountExistsState(
    CHAIN_ID_SOLANA,
    fromMint,
    fromTokenAccount
  );
  const {
    associatedAccountExists: toTokenAccountExists,
    setAssociatedAccountExists: setToTokenAccountExists,
  } = useAssociatedAccountExistsState(CHAIN_ID_SOLANA, toMint, toTokenAccount);

  const [toCustodyAddress, setToCustodyAddress] = useState<string | undefined>(
    undefined
  );
  const [toCustodyBalance, setToCustodyBalance] = useState<string | undefined>(
    undefined
  );

  const [migrationAmount, setMigrationAmount] = useState("");
  const [migrationIsProcessing, setMigrationIsProcessing] = useState(false);
  const [error, setError] = useState("");
  const [transaction, setTransaction] = useState<string | null>(null);

  /* Effects
   */
  useEffect(() => {
    getDecimals(connection, fromMint, setFromMintDecimals);
  }, [connection, fromMint]);

  //Retrieve user balance when fromTokenAccount changes
  useEffect(() => {
    // TODO: cancellable
    if (fromTokenAccount && fromTokenAccountExists) {
      getBalance(connection, fromTokenAccount, setFromTokenAccountBalance);
    } else {
      setFromTokenAccountBalance(undefined);
    }
  }, [
    connection,
    fromTokenAccountExists,
    fromTokenAccount,
    setFromTokenAccountBalance,
  ]);

  useEffect(() => {
    // TODO: cancellable
    if (toTokenAccount && toTokenAccountExists) {
      getBalance(connection, toTokenAccount, setToTokenAccountBalance);
    } else {
      setToTokenAccountBalance(undefined);
    }
  }, [
    connection,
    toTokenAccountExists,
    toTokenAccount,
    setFromTokenAccountBalance,
  ]);

  useEffect(() => {
    // TODO: cancellable
    if (toCustodyAddress) {
      getBalance(connection, toCustodyAddress, setToCustodyBalance);
    } else {
      setToCustodyBalance(undefined);
    }
  }, [connection, toCustodyAddress, setToCustodyBalance]);

  //Retrieve pool address on selectedTokens change
  useEffect(() => {
    if (toMint && fromMint) {
      setPoolAddress("");
      setPoolExists(undefined);
      getPoolAddress(MIGRATION_PROGRAM_ADDRESS, fromMint, toMint).then(
        (result) => {
          const key = new PublicKey(result).toString();
          setPoolAddress(key);
        },
        (error) => console.log("Could not calculate pool address.")
      );
    }
  }, [toMint, fromMint, setPoolAddress]);

  //Retrieve the poolAccount every time the pool address changes.
  useEffect(() => {
    if (poolAddress) {
      setPoolExists(undefined);
      try {
        getMultipleAccounts(
          connection,
          [new PublicKey(poolAddress)],
          "confirmed"
        ).then((result) => {
          if (result.length && result[0] !== null) {
            setPoolExists(true);
          } else if (result.length && result[0] === null) {
            setPoolExists(false);
            setError("There is no swap pool for this token.");
          } else {
            setError(
              "unexpected error in fetching pool address. Please reload and try again"
            );
          }
        });
      } catch (e) {
        setError("Could not fetch pool address");
      }
    }
  }, [connection, poolAddress]);

  //Set relevant information derived from poolAddress
  useEffect(() => {
    if (poolAddress) {
      getToCustodyAddress(MIGRATION_PROGRAM_ADDRESS, poolAddress)
        .then((result: any) =>
          setToCustodyAddress(new PublicKey(result).toString())
        )
        .catch((e) => {
          setToCustodyAddress(undefined);
        });
    } else {
      setToCustodyAddress(undefined);
    }
  }, [poolAddress]);

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
  /*
    End effects
    */

  const migrateTokens = useCallback(async () => {
    try {
      setError("");
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
          setMigrationIsProcessing(false);
          setTransaction(transaction);
        },
        (error) => {
          console.log(error);
          setError("Could not complete the migrateTokens transaction.");
          setMigrationIsProcessing(false);
        }
      );
    } catch (e) {
      console.log(e);
      setError("Could not complete the migrateTokens transaction.");
      setMigrationIsProcessing(false);
    }
  }, [
    connection,
    fromMint,
    fromTokenAccount,
    migrationAmount,
    toMint,
    toTokenAccount,
    wallet,
    fromMintDecimals,
  ]);

  const fromParse = (amount: string) => {
    try {
      return parseUnits(amount, fromMintDecimals).toBigInt();
    } catch (e) {
      return BigInt(0);
    }
  };

  const hasRequisiteData = fromMint && toMint && poolAddress && poolExists;
  const accountsReady =
    fromTokenAccountExists && toTokenAccountExists && poolExists;
  const amountGreaterThanZero = fromParse(migrationAmount) > BigInt(0);
  const sufficientFromTokens =
    fromTokenAccountBalance &&
    migrationAmount &&
    fromParse(migrationAmount) <= fromParse(fromTokenAccountBalance);
  const sufficientPoolBalance =
    toCustodyBalance &&
    migrationAmount &&
    parseFloat(migrationAmount) <= parseFloat(toCustodyBalance);

  const isReadyToTransfer =
    isReady &&
    amountGreaterThanZero &&
    sufficientFromTokens &&
    sufficientPoolBalance &&
    accountsReady &&
    hasRequisiteData;

  const getNotReadyCause = () => {
    if (!fromMint || !toMint || !poolAddress || !poolExists) {
      return "This asset is not supported.";
    } else if (!isReady) {
      return "Wallet is not connected.";
    } else if (!toTokenAccountExists || !fromTokenAccountExists) {
      return "You have not created the necessary token accounts.";
    } else if (!migrationAmount) {
      return "Enter an amount to transfer.";
    } else if (!amountGreaterThanZero) {
      return "Enter an amount greater than zero.";
    } else if (!sufficientFromTokens) {
      return "There are not sufficient funds in your wallet for this transfer.";
    } else if (!sufficientPoolBalance) {
      return "There are not sufficient funds in the pool for this transfer.";
    } else {
      return "";
    }
  };

  const handleAmountChange = useCallback(
    (event) => setMigrationAmount(event.target.value),
    [setMigrationAmount]
  );
  const handleMaxClick = useCallback(() => {
    if (fromTokenAccountBalance) {
      setMigrationAmount(fromTokenAccountBalance);
    }
  }, [fromTokenAccountBalance]);

  const getMetadata = (address: string) => {
    const tokenMapItem = solanaTokenMap.data?.find(
      (x) => x.address === address
    );
    const metaplexItem = metaplexData.data?.get(address);

    return {
      symbol: tokenMapItem?.symbol || metaplexItem?.data?.symbol || undefined,
      name: tokenMapItem?.name || metaplexItem?.data?.name || undefined,
      logo: tokenMapItem?.logoURI || metaplexItem?.data?.uri || undefined,
    };
  };

  const toMetadata = getMetadata(toMint);
  const fromMetadata = getMetadata(fromMint);

  const toMintPretty = (
    <SmartAddress
      chainId={CHAIN_ID_SOLANA}
      address={toMint}
      symbol={toMetadata?.symbol}
      tokenName={toMetadata?.name}
    />
  );
  const fromMintPretty = (
    <SmartAddress
      chainId={CHAIN_ID_SOLANA}
      address={fromMint}
      symbol={fromMetadata?.symbol}
      tokenName={fromMetadata?.name}
    />
  );

  return (
    <div>
      <SolanaWalletKey />
      <div className={classes.spacer} />
      {fromTokenAccount && toTokenAccount ? (
        <>
          <Typography variant="body2" component="div">
            <span>This will migrate</span>
            {fromMintPretty}
            <span>tokens in this account:</span>
          </Typography>
          <Typography variant="h5">
            <SmartAddress
              address={fromTokenAccount}
              chainId={CHAIN_ID_SOLANA}
            />
            {`(Balance: ${fromTokenAccountBalance}${
              fromMetadata.symbol && " " + fromMetadata.symbol
            })`}
          </Typography>
          <div className={classes.spacer} />
          <Typography variant="body2" component="div">
            <span>into </span>
            {toMintPretty}
            <span> tokens in this account:</span>
          </Typography>
          <Typography
            variant="h5"
            color={toTokenAccountExists ? "textPrimary" : "textSecondary"}
          >
            <SmartAddress address={toTokenAccount} chainId={CHAIN_ID_SOLANA} />
            <span>
              {toTokenAccountExists
                ? ` (Balance: ${toTokenAccountBalance}${
                    (toMetadata.symbol && " " + toMetadata.symbol) || ""
                  })`
                : " (Not created yet)"}
            </span>
          </Typography>
          <SolanaCreateAssociatedAddress
            mintAddress={toMint}
            readableTargetAddress={toTokenAccount}
            associatedAccountExists={toTokenAccountExists}
            setAssociatedAccountExists={setToTokenAccountExists}
          />
          {poolAddress && toCustodyAddress && toCustodyBalance ? (
            <>
              <div className={classes.spacer} />
              <Typography variant="body2" component="div">
                <span>Using pool </span>
                <SmartAddress address={poolAddress} chainId={CHAIN_ID_SOLANA} />
                <span> holding tokens in this account:</span>
              </Typography>
              <Typography variant="h5">
                <SmartAddress
                  address={toCustodyAddress}
                  chainId={CHAIN_ID_SOLANA}
                />
                <span>{` (Balance: ${toCustodyBalance}${
                  toMetadata.symbol && " " + toMetadata.symbol
                })`}</span>
              </Typography>
            </>
          ) : null}
        </>
      ) : null}
      <div className={classes.spacer} />
      <NumberTextField
        variant="outlined"
        value={migrationAmount}
        onChange={handleAmountChange}
        label={"Amount"}
        disabled={!!migrationIsProcessing || !!transaction}
        onMaxClick={fromTokenAccountBalance ? handleMaxClick : undefined}
      />

      {!transaction && (
        <ButtonWithLoader
          disabled={!isReadyToTransfer || migrationIsProcessing}
          showLoader={migrationIsProcessing}
          onClick={migrateTokens}
        >
          {migrationAmount && isReadyToTransfer
            ? "Migrate " + migrationAmount + " Tokens"
            : "Migrate"}
        </ButtonWithLoader>
      )}
      {(error || !isReadyToTransfer) && (
        <Typography color="error">{error || getNotReadyCause()}</Typography>
      )}
      {transaction ? (
        <>
          <Typography>
            Successfully migrated your tokens! They will be available once this
            transaction confirms.
          </Typography>
          <ShowTx
            tx={{ id: transaction, block: 1 }}
            chainId={CHAIN_ID_SOLANA}
          />
        </>
      ) : null}
    </div>
  );
}
