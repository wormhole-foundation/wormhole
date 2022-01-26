import { CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import {
  CircularProgress,
  Container,
  makeStyles,
  Paper,
  Typography,
} from "@material-ui/core";
import ArrowRightAltIcon from "@material-ui/icons/ArrowRightAlt";
import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import {
  AccountInfo,
  Connection,
  ParsedAccountData,
  PublicKey,
} from "@solana/web3.js";
import { useCallback, useEffect, useMemo, useState } from "react";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import useSolanaMigratorInformation from "../../hooks/useSolanaMigratorInformation";
import { COLORS } from "../../muiTheme";
import {
  CHAINS_BY_ID,
  getMigrationAssetMap,
  SOLANA_HOST,
} from "../../utils/consts";
import ButtonWithLoader from "../ButtonWithLoader";
import HeaderText from "../HeaderText";
import ShowTx from "../ShowTx";
import SmartAddress from "../SmartAddress";
import SolanaCreateAssociatedAddress from "../SolanaCreateAssociatedAddress";
import SolanaWalletKey from "../SolanaWalletKey";

const useStyles = makeStyles((theme) => ({
  spacer: {
    height: "2rem",
  },
  containerDiv: {
    textAlign: "center",
    padding: theme.spacing(2),
  },
  centered: {
    textAlign: "center",
  },
  lineItem: {
    display: "flex",
    flexWrap: "nowrap",
    justifyContent: "space-between",
    "& > *": {
      alignSelf: "flex-start",
      width: "max-content",
    },
  },
  flexGrow: {
    flewGrow: 1,
  },
  mainPaper: {
    backgroundColor: COLORS.whiteWithTransparency,
    textAlign: "center",
    padding: "2rem",
    "& > h, p ": {
      margin: ".5rem",
    },
  },
  hidden: {
    display: "none",
  },
  divider: {
    margin: "2rem 0rem 2rem 0rem",
  },
  balance: {
    display: "inline-block",
  },
  convertButton: {
    alignSelf: "flex-end",
  },
}));

function SolanaMigrationLineItem({
  migratorInfo,
  onLoadComplete,
}: {
  migratorInfo: DefaultAssociatedTokenAccountInfo;
  onLoadComplete: () => void;
}) {
  const classes = useStyles();
  const poolInfo = useSolanaMigratorInformation(
    migratorInfo.fromMintKey,
    migratorInfo.toMintKey,
    migratorInfo.defaultFromTokenAccount
  );

  const [migrationIsProcessing, setMigrationIsProcessing] = useState(false);
  const [transaction, setTransaction] = useState("");
  const [migrationError, setMigrationError] = useState("");

  const handleMigrateClick = useCallback(() => {
    if (!poolInfo.data) {
      return;
    }
    setMigrationIsProcessing(true);
    setMigrationError("");
    poolInfo.data
      .migrateTokens(poolInfo.data.fromAssociatedTokenAccountBalance)
      .then((result) => {
        setMigrationIsProcessing(false);
        setTransaction(result);
      })
      .catch((e) => {
        setMigrationError("Unable to perform migration.");
        setMigrationIsProcessing(false);
      });
  }, [poolInfo.data]);

  const precheckError =
    poolInfo.data &&
    poolInfo.data.getNotReadyCause(
      poolInfo.data.fromAssociatedTokenAccountBalance
    );

  useEffect(() => {
    if (poolInfo.data || poolInfo.error) {
      onLoadComplete();
    }
  }, [poolInfo, onLoadComplete]);

  if (!poolInfo.data) {
    return (
      <div className={classes.centered}>
        <div>
          <Typography variant="body2" color="textSecondary">
            Failed to load migration information for token
          </Typography>
          <SmartAddress
            chainId={CHAIN_ID_SOLANA}
            address={migratorInfo.fromMintKey}
          />
        </div>
      </div>
    );
  } else if (transaction) {
    return (
      <div className={classes.centered}>
        <div>
          <Typography variant="body2" color="textSecondary">
            Successfully migrated your tokens. They will become available once
            this transaction confirms.
          </Typography>
          <ShowTx
            chainId={CHAIN_ID_SOLANA}
            tx={{ id: transaction, block: 1 }}
          />
        </div>
      </div>
    );
  } else {
    return (
      <div className={classes.lineItem}>
        <div>
          <Typography variant="body2" color="textSecondary">
            Current Token
          </Typography>
          <Typography className={classes.balance}>
            {poolInfo.data.fromAssociatedTokenAccountBalance}
          </Typography>
          <SmartAddress
            chainId={CHAIN_ID_SOLANA}
            address={poolInfo.data.fromAssociatedTokenAccount}
            symbol={poolInfo.data.fromSymbol || undefined}
            tokenName={poolInfo.data.fromName || undefined}
          />
        </div>
        <div>
          <Typography variant="body2" color="textSecondary">
            will become
          </Typography>
          <ArrowRightAltIcon fontSize="large" />
        </div>
        <div>
          <Typography variant="body2" color="textSecondary">
            Wormhole Token
          </Typography>
          <Typography className={classes.balance}>
            {poolInfo.data.fromAssociatedTokenAccountBalance}
          </Typography>
          <SmartAddress
            chainId={CHAIN_ID_SOLANA}
            address={poolInfo.data.toAssociatedTokenAccount}
            symbol={poolInfo.data.toSymbol || undefined}
            tokenName={poolInfo.data.toName || undefined}
          />
        </div>
        {!poolInfo.data.toAssociatedTokenAccountExists ? (
          <div className={classes.convertButton}>
            <SolanaCreateAssociatedAddress
              mintAddress={migratorInfo.toMintKey}
              readableTargetAddress={poolInfo.data?.toAssociatedTokenAccount}
              associatedAccountExists={
                poolInfo.data.toAssociatedTokenAccountExists
              }
              setAssociatedAccountExists={poolInfo.data.setToTokenAccountExists}
            />
          </div>
        ) : (
          <div className={classes.convertButton}>
            <ButtonWithLoader
              showLoader={migrationIsProcessing}
              onClick={handleMigrateClick}
              error={
                poolInfo.error
                  ? poolInfo.error
                  : migrationError
                  ? migrationError
                  : precheckError
                  ? precheckError
                  : ""
              }
              disabled={
                !!poolInfo.error || !!precheckError || migrationIsProcessing
              }
            >
              Convert
            </ButtonWithLoader>
          </div>
        )}
      </div>
    );
  }
}

type DefaultAssociatedTokenAccountInfo = {
  fromMintKey: string;
  toMintKey: string;
  defaultFromTokenAccount: string;
  fromAccountInfo: AccountInfo<ParsedAccountData> | null;
};

const getTokenBalances = async (
  walletAddress: string,
  migrationMap: Map<string, string>
): Promise<DefaultAssociatedTokenAccountInfo[]> => {
  try {
    const connection = new Connection(SOLANA_HOST);
    const output: DefaultAssociatedTokenAccountInfo[] = [];
    const tokenAccounts = await connection.getParsedTokenAccountsByOwner(
      new PublicKey(walletAddress),
      { programId: TOKEN_PROGRAM_ID },
      "confirmed"
    );
    tokenAccounts.value.forEach((item) => {
      if (
        item.account != null &&
        item.account.data?.parsed?.info?.tokenAmount?.uiAmountString &&
        item.account.data?.parsed.info?.tokenAmount?.amount !== "0"
      ) {
        const fromMintKey = item.account.data.parsed.info.mint;
        const toMintKey = migrationMap.get(fromMintKey);
        if (toMintKey) {
          output.push({
            fromMintKey,
            toMintKey: toMintKey,
            defaultFromTokenAccount: item.pubkey.toString(),
            fromAccountInfo: item.account,
          });
        }
      }
    });

    return output;
  } catch (e) {
    console.error(e);
    return Promise.reject("Unable to retrieve token balances.");
  }
};

export default function SolanaQuickMigrate() {
  const chainId = CHAIN_ID_SOLANA;
  const classes = useStyles();
  const { isReady, walletAddress } = useIsWalletReady(chainId);
  const migrationMap = useMemo(() => getMigrationAssetMap(chainId), [chainId]);
  const [migrators, setMigrators] = useState<
    DefaultAssociatedTokenAccountInfo[] | null
  >(null);
  const [migratorsError, setMigratorsError] = useState("");
  const [migratorsLoading, setMigratorsLoading] = useState(false);

  //This is for a callback into the line items, so a loader can be displayed while
  //they are loading
  //TODO don't just swallow loading errors.
  const [migratorsFinishedLoading, setMigratorsFinishedLoading] = useState(0);
  const reportLoadComplete = useCallback(() => {
    setMigratorsFinishedLoading((prevState) => prevState + 1);
  }, []);
  const isLoading =
    migratorsLoading ||
    (migrators &&
      migrators.length &&
      migratorsFinishedLoading < migrators.length);

  useEffect(() => {
    if (isReady && walletAddress) {
      let cancelled = false;
      setMigratorsLoading(true);
      setMigratorsError("");
      getTokenBalances(walletAddress, migrationMap).then(
        (result) => {
          if (!cancelled) {
            setMigratorsFinishedLoading(0);
            setMigrators(result.filter((x) => x.fromAccountInfo && x));
            setMigratorsLoading(false);
          }
        },
        (error) => {
          if (!cancelled) {
            setMigratorsLoading(false);
            setMigratorsError(
              "Failed to retrieve available token information."
            );
          }
        }
      );

      return () => {
        cancelled = true;
      };
    }
  }, [isReady, walletAddress, migrationMap]);

  const hasEligibleAssets = migrators && migrators.length > 0;
  const chainName = CHAINS_BY_ID[chainId]?.name;

  const content = (
    <div className={classes.containerDiv}>
      <Typography variant="h5">
        {`This page allows you to convert certain wrapped tokens ${
          chainName ? "on " + chainName : ""
        } into
        Wormhole V2 tokens.`}
      </Typography>
      <SolanaWalletKey />
      {!isReady ? (
        <Typography variant="body1">Please connect your wallet.</Typography>
      ) : migratorsError ? (
        <Typography variant="h6">{migratorsError}</Typography>
      ) : (
        <>
          <div className={classes.spacer} />
          <CircularProgress className={isLoading ? "" : classes.hidden} />
          <div className={!isLoading ? "" : classes.hidden}>
            <Typography>
              {hasEligibleAssets
                ? "You have some assets that are eligible for migration! Click the 'Convert' button to swap them for Wormhole tokens."
                : "You don't have any assets eligible for migration."}
            </Typography>
            <div className={classes.spacer} />
            {migrators?.map((info) => {
              return (
                <SolanaMigrationLineItem
                  migratorInfo={info}
                  onLoadComplete={reportLoadComplete}
                />
              );
            })}
          </div>
        </>
      )}
    </div>
  );

  return (
    <Container maxWidth="md">
      <HeaderText
        white
        subtitle="Convert assets from other bridges to Wormhole V2 tokens"
      >
        Migrate Assets
      </HeaderText>
      <Paper className={classes.mainPaper}>{content}</Paper>
    </Container>
  );
}
