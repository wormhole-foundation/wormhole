import { ChainId, TokenImplementation__factory } from "@certusone/wormhole-sdk";
import { Signer } from "@ethersproject/abstract-signer";
import { getAddress } from "@ethersproject/address";
import { BigNumber } from "@ethersproject/bignumber";
import {
  CircularProgress,
  Container,
  makeStyles,
  Paper,
  Typography,
} from "@material-ui/core";
import ArrowRightAltIcon from "@material-ui/icons/ArrowRightAlt";
import { Alert } from "@material-ui/lab";
import { parseUnits } from "ethers/lib/utils";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import useEthereumMigratorInformation from "../../hooks/useEthereumMigratorInformation";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import { COLORS } from "../../muiTheme";
import { CHAINS_BY_ID, getMigrationAssetMap } from "../../utils/consts";
import ButtonWithLoader from "../ButtonWithLoader";
import EthereumSignerKey from "../EthereumSignerKey";
import HeaderText from "../HeaderText";
import ShowTx from "../ShowTx";
import SmartAddress from "../SmartAddress";

const useStyles = makeStyles((theme) => ({
  spacer: {
    height: "2rem",
  },
  containerDiv: {
    textAlign: "center",
    padding: theme.spacing(2),
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

//TODO move elsewhere
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

function EvmMigrationLineItem({
  chainId,
  migratorAddress,
  onLoadComplete,
}: {
  chainId: ChainId;
  migratorAddress: string;
  onLoadComplete: () => void;
}) {
  const classes = useStyles();
  const { enqueueSnackbar } = useSnackbar();
  const { signer, signerAddress } = useEthereumProvider();
  const poolInfo = useEthereumMigratorInformation(
    migratorAddress,
    signer,
    signerAddress,
    false
  );
  const [loaded, setLoaded] = useState(false);
  const [migrationIsProcessing, setMigrationIsProcessing] = useState(false);
  const [transaction, setTransaction] = useState("");
  const [error, setError] = useState("");
  const fromSymbol = poolInfo?.data?.fromSymbol;
  const toSymbol = poolInfo?.data?.toSymbol;

  const sufficientPoolBalance =
    poolInfo.data &&
    compareWithDecimalOffset(
      poolInfo.data.fromWalletBalance,
      poolInfo.data.fromDecimals,
      poolInfo.data.toPoolBalance,
      poolInfo.data.toDecimals
    ) !== 1;

  useEffect(() => {
    if (!loaded && (poolInfo.data || poolInfo.error)) {
      onLoadComplete();
      setLoaded(true);
    }
  }, [loaded, poolInfo, onLoadComplete]);

  //TODO use transaction loader
  const migrateTokens = useCallback(async () => {
    if (!poolInfo.data) {
      enqueueSnackbar(null, {
        content: <Alert severity="error">Could not migrate the tokens.</Alert>,
      }); //Should never be hit
      return;
    }
    try {
      const migrationAmountAbs = parseUnits(
        poolInfo.data.fromWalletBalance,
        poolInfo.data.fromDecimals
      );
      setMigrationIsProcessing(true);
      await poolInfo.data.fromToken.approve(
        poolInfo.data.migrator.address,
        migrationAmountAbs
      );
      const transaction = await poolInfo.data.migrator.migrate(
        migrationAmountAbs
      );
      await transaction.wait();
      setTransaction(transaction.hash);
      enqueueSnackbar(null, {
        content: (
          <Alert severity="success">Successfully migrated the tokens.</Alert>
        ),
      });
      setMigrationIsProcessing(false);
    } catch (e) {
      console.error(e);
      enqueueSnackbar(null, {
        content: <Alert severity="error">Could not migrate the tokens.</Alert>,
      });
      setMigrationIsProcessing(false);
      setError("Failed to send the transaction.");
    }
  }, [poolInfo.data, enqueueSnackbar]);

  if (!poolInfo.data) {
    return null;
  } else if (transaction) {
    return (
      <div className={classes.lineItem}>
        <div>
          <Typography variant="body2" color="textSecondary">
            Successfully migrated your tokens. They will become available once
            this transaction confirms.
          </Typography>
          <ShowTx chainId={chainId} tx={{ id: transaction, block: 1 }} />
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
            {poolInfo.data.fromWalletBalance}
          </Typography>
          <SmartAddress
            chainId={chainId}
            address={poolInfo.data.fromAddress}
            symbol={fromSymbol || undefined}
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
            {poolInfo.data.fromWalletBalance}
          </Typography>
          <SmartAddress
            chainId={chainId}
            address={poolInfo.data.toAddress}
            symbol={toSymbol || undefined}
          />
        </div>
        <div className={classes.convertButton}>
          <ButtonWithLoader
            showLoader={migrationIsProcessing}
            onClick={migrateTokens}
            error={
              error
                ? error
                : !sufficientPoolBalance
                ? "The swap pool has insufficient funds."
                : ""
            }
            disabled={!sufficientPoolBalance || migrationIsProcessing}
          >
            Convert
          </ButtonWithLoader>
        </div>
      </div>
    );
  }
}

const getAddressBalances = async (
  signer: Signer,
  signerAddress: string,
  addresses: string[]
): Promise<Map<string, BigNumber | null>> => {
  try {
    const promises: Promise<any>[] = [];
    const output = new Map<string, BigNumber | null>();
    addresses.forEach((address) => {
      const factory = TokenImplementation__factory.connect(address, signer);
      promises.push(
        factory.balanceOf(signerAddress).then(
          (result) => {
            output.set(address, result);
          },
          (error) => {
            output.set(address, null);
          }
        )
      );
    });
    await Promise.all(promises);
    return output;
  } catch (e) {
    return Promise.reject("Unable to retrieve token balances.");
  }
};

export default function EvmQuickMigrate({ chainId }: { chainId: ChainId }) {
  const classes = useStyles();
  const { signer, signerAddress } = useEthereumProvider();
  const { isReady } = useIsWalletReady(chainId);
  const migrationMap = useMemo(() => getMigrationAssetMap(chainId), [chainId]);
  const eligibleTokens = useMemo(
    () => Array.from(migrationMap.keys()),
    [migrationMap]
  );
  const [migrators, setMigrators] = useState<string[] | null>(null);
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
    if (isReady && signer && signerAddress) {
      let cancelled = false;
      setMigratorsLoading(true);
      setMigratorsError("");
      getAddressBalances(signer, signerAddress, eligibleTokens).then(
        (result) => {
          if (!cancelled) {
            const migratorAddresses = [];
            for (const tokenAddress of result.keys()) {
              if (result.get(tokenAddress) && result.get(tokenAddress)?.gt(0)) {
                const migratorAddress = migrationMap.get(
                  getAddress(tokenAddress)
                );
                if (migratorAddress) {
                  migratorAddresses.push(migratorAddress);
                }
              }
            }
            setMigratorsFinishedLoading(0);
            setMigrators(migratorAddresses);
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
  }, [isReady, signer, signerAddress, eligibleTokens, migrationMap]);

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
      <EthereumSignerKey />
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
            {migrators?.map((address) => {
              return (
                <EvmMigrationLineItem
                  key={address}
                  chainId={chainId}
                  migratorAddress={address}
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
