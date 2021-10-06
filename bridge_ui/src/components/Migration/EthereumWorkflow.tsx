import { CHAIN_ID_ETH } from "@certusone/wormhole-sdk";
import {
  CircularProgress,
  makeStyles,
  TextField,
  Typography,
} from "@material-ui/core";
import { parseUnits } from "ethers/lib/utils";
import { useSnackbar } from "notistack";
import { useCallback, useState } from "react";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import useEthereumMigratorInformation from "../../hooks/useEthereumMigratorInformation";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import ButtonWithLoader from "../ButtonWithLoader";
import EthereumSignerKey from "../EthereumSignerKey";
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
}));

export default function EthereumWorkflow({
  migratorAddress,
}: {
  migratorAddress: string;
}) {
  const classes = useStyles();
  const { enqueueSnackbar } = useSnackbar();
  const { signer, signerAddress } = useEthereumProvider();
  const { isReady } = useIsWalletReady(CHAIN_ID_ETH);
  const [toggleRefresh, setToggleRefresh] = useState(false);
  const forceRefresh = useCallback(
    () => setToggleRefresh((prevState) => !prevState),
    []
  );
  const poolInfo = useEthereumMigratorInformation(
    migratorAddress,
    signer,
    signerAddress,
    toggleRefresh
  );

  const [migrationAmount, setMigrationAmount] = useState("");
  const [migrationIsProcessing, setMigrationIsProcessing] = useState(false);
  const [error, setError] = useState("");
  const [transaction, setTransaction] = useState<string | null>(null);

  const fromParse = (amount: string) => {
    try {
      if (!poolInfo.data?.fromDecimals || !migrationAmount) {
        return BigInt(0);
      }
      return parseUnits(amount, poolInfo.data.fromDecimals).toBigInt();
    } catch (e) {
      return BigInt(0);
    }
  };

  const hasRequisiteData = poolInfo.data;
  const amountGreaterThanZero = fromParse(migrationAmount) > BigInt(0);
  const sufficientFromTokens =
    poolInfo.data?.fromWalletBalance &&
    migrationAmount &&
    fromParse(migrationAmount) <= fromParse(poolInfo.data.fromWalletBalance);
  const sufficientPoolBalance =
    poolInfo.data?.toPoolBalance &&
    migrationAmount &&
    parseFloat(migrationAmount) <= parseFloat(poolInfo.data.toPoolBalance);

  const isReadyToTransfer =
    isReady &&
    amountGreaterThanZero &&
    sufficientFromTokens &&
    sufficientPoolBalance &&
    hasRequisiteData;

  const getNotReadyCause = () => {
    if (!isReady) {
      return "Connect your wallet to proceed.";
    } else if (poolInfo.error) {
      return "Unable to retrieve necessary information. This asset may not be supported.";
    } else if (!migrationAmount) {
      return "Enter an amount to transfer.";
    } else if (!amountGreaterThanZero) {
      return "The transfer amount must be greater than zero.";
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

  const migrateTokens = useCallback(async () => {
    if (!poolInfo.data) {
      enqueueSnackbar("Could not migrate the tokens.", { variant: "error" }); //Should never be hit
      return;
    }
    try {
      setMigrationIsProcessing(true);
      setError("");
      await poolInfo.data.fromToken.approve(
        poolInfo.data.migrator.address,
        parseUnits(migrationAmount, poolInfo.data.fromDecimals)
      );
      const transaction = await poolInfo.data.migrator.migrate(
        parseUnits(migrationAmount, poolInfo.data.fromDecimals)
      );
      await transaction.wait();
      setTransaction(transaction.hash);
      forceRefresh();
      enqueueSnackbar(`Successfully migrated the tokens.`, {
        variant: "success",
      });
      setMigrationIsProcessing(false);
    } catch (e) {
      console.error(e);
      enqueueSnackbar("Could not migrate the tokens.", { variant: "error" });
      setMigrationIsProcessing(false);
      setError("Failed to send the transaction.");
    }
  }, [poolInfo.data, migrationAmount, enqueueSnackbar, forceRefresh]);

  //TODO tokenName
  const toTokenPretty = (
    <SmartAddress
      chainId={CHAIN_ID_ETH}
      address={poolInfo.data?.toAddress}
      symbol={poolInfo.data?.toSymbol}
    />
  );
  const fromTokenPretty = (
    <SmartAddress
      chainId={CHAIN_ID_ETH}
      address={poolInfo.data?.fromAddress}
      symbol={poolInfo.data?.fromSymbol}
    />
  );
  const poolPretty = (
    <SmartAddress chainId={CHAIN_ID_ETH} address={poolInfo.data?.poolAddress} />
  );

  const fatalError = poolInfo.error
    ? "Unable to retrieve necessary information. This asset may not be supported."
    : null;

  const explainerContent = (
    <div>
      <Typography>This action will convert</Typography>
      <Typography variant="h6">
        {fromTokenPretty}{" "}
        {`(Balance: ${poolInfo.data?.fromWalletBalance || ""})`}
      </Typography>
      <div className={classes.spacer} />
      <Typography>to</Typography>
      <Typography variant="h6">
        {toTokenPretty} {`(Balance: ${poolInfo.data?.toWalletBalance || ""})`}
      </Typography>
      <div className={classes.spacer} />
      <Typography>Utilizing this pool</Typography>
      <Typography variant="h6">
        {poolPretty} {`(Balance: ${poolInfo.data?.toPoolBalance || ""})`}
      </Typography>
    </div>
  );

  const mainWorkflow = (
    <>
      {explainerContent}
      <div className={classes.spacer} />
      <TextField
        variant="outlined"
        value={migrationAmount}
        type="number"
        onChange={handleAmountChange}
        label={"Amount"}
        disabled={!!migrationIsProcessing || !!transaction}
      ></TextField>

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
          <ShowTx tx={{ id: transaction, block: 1 }} chainId={CHAIN_ID_ETH} />
        </>
      ) : null}
    </>
  );

  return (
    <div className={classes.containerDiv}>
      <EthereumSignerKey />
      {!isReady ? (
        <Typography variant="body1">Please connect your wallet.</Typography>
      ) : poolInfo.isLoading ? (
        <CircularProgress />
      ) : fatalError ? (
        <Typography variant="h6">{fatalError}</Typography>
      ) : (
        mainWorkflow
      )}
    </div>
  );
}
