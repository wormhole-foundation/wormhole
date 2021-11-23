import { CHAIN_ID_ETH } from "@certusone/wormhole-sdk";
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
//import { pool_address } from "@certusone/wormhole-sdk/lib/esm/solana/migration/wormhole_migration";
import { parseUnits } from "ethers/lib/utils";
import { useCallback, useState } from "react";
import EthereumSignerKey from "../components/EthereumSignerKey";
import LogWatcher from "../components/LogWatcher";
import SmartAddress from "../components/SmartAddress";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useLogger } from "../contexts/Logger";
import useEthereumMigratorInformation from "../hooks/useEthereumMigratorInformation";
import { compareWithDecimalOffset } from "./Main";

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

function MigrateEthereum() {
  const classes = useStyles();
  const { signer, signerAddress } = useEthereumProvider();
  const { log } = useLogger();

  const [migratorAddress, setMigratorAddress] = useState("");
  const [refresher, setRefresher] = useState(false);
  const forceRefresh = useCallback(() => {
    setRefresher((prevState) => !prevState);
  }, []);
  const poolInfo = useEthereumMigratorInformation(
    migratorAddress,
    signer,
    signerAddress,
    refresher
  );
  const info = poolInfo.data;

  const [liquidityAmount, setLiquidityAmount] = useState("");
  const [removeLiquidityAmount, setRemoveLiquidityAmount] = useState("");
  const [migrationAmount, setMigrationAmount] = useState("");
  const [redeemAmount, setRedeemAmount] = useState("");

  const [liquidityIsProcessing, setLiquidityIsProcessing] = useState(false);
  const [removeLiquidityIsProcessing, setRemoveLiquidityIsProcessing] =
    useState(false);
  const [migrationIsProcessing, setMigrationIsProcessing] = useState(false);
  const [redeemIsProcessing, setRedeemIsProcessing] = useState(false);

  const addLiquidity = useCallback(async () => {
    if (!info) {
      return;
    }
    try {
      setLiquidityIsProcessing(true);
      await info.toToken.approve(
        info.migrator.address,
        parseUnits(liquidityAmount, info.toDecimals)
      );
      const transaction = await info.migrator.add(
        parseUnits(liquidityAmount, info.toDecimals)
      );
      await transaction.wait();
      forceRefresh();
      log(`Successfully added liquidity to the pool.`, "success");
      setLiquidityIsProcessing(false);
    } catch (e) {
      console.error(e);
      log(`Could not add liquidity to the pool.`, "error");
      setLiquidityIsProcessing(false);
    }
  }, [info, liquidityAmount, log, forceRefresh]);

  const removeLiquidity = useCallback(async () => {
    if (!info) {
      return;
    }
    try {
      setRemoveLiquidityIsProcessing(true);
      const transaction = await info.migrator.remove(
        parseUnits(removeLiquidityAmount, info.sharesDecimals)
      );
      await transaction.wait();
      forceRefresh();
      log(`Successfully removed liquidity from the pool.`, "success");
      setRemoveLiquidityIsProcessing(false);
    } catch (e) {
      console.error(e);
      log(`Could not remove liquidity from the pool.`, "error");
      setRemoveLiquidityIsProcessing(false);
    }
  }, [info, removeLiquidityAmount, log, forceRefresh]);

  const migrateTokens = useCallback(async () => {
    if (!info) {
      return;
    }
    try {
      setMigrationIsProcessing(true);
      await info.fromToken.approve(
        info.migrator.address,
        parseUnits(migrationAmount, info.fromDecimals)
      );
      const transaction = await info.migrator.migrate(
        parseUnits(migrationAmount, info.fromDecimals)
      );
      await transaction.wait();
      forceRefresh();
      log(`Successfully migrated tokens.`, "success");
      setMigrationIsProcessing(false);
    } catch (e) {
      console.error(e);
      log(`Could not migrate the tokens.`, "error");
      setMigrationIsProcessing(false);
    }
  }, [info, migrationAmount, log, forceRefresh]);

  const redeemShares = useCallback(async () => {
    if (!info) {
      return;
    }
    try {
      setRedeemIsProcessing(true);
      const transaction = await info.migrator.claim(
        parseUnits(redeemAmount, info.sharesDecimals)
      );
      await transaction.wait();
      forceRefresh();
      log(`Successfully redeemed shares.`, "success");
      setRedeemIsProcessing(false);
    } catch (e) {
      console.error(e);
      log(`Could not redeem shares.`, "error");
      setRedeemIsProcessing(false);
    }
  }, [info, redeemAmount, log, forceRefresh]);

  const addToTokensInWallet =
    info &&
    liquidityAmount &&
    compareWithDecimalOffset(
      liquidityAmount,
      info.toDecimals,
      info.toWalletBalance,
      info.toDecimals
    ) !== 1;
  const addLiquidityIsReady = addToTokensInWallet;
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

  const removeToTokensInPool =
    info &&
    removeLiquidityAmount &&
    compareWithDecimalOffset(
      removeLiquidityAmount,
      info.sharesDecimals,
      info.toPoolBalance,
      info.toDecimals
    ) !== 1;
  const removeShareTokensInWallet =
    info &&
    removeLiquidityAmount &&
    compareWithDecimalOffset(
      removeLiquidityAmount,
      info.sharesDecimals,
      info.walletSharesBalance,
      info.sharesDecimals
    ) !== 1;
  const removeLiquidityIsReady =
    removeShareTokensInWallet && removeToTokensInPool;
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

  const migrateToTokensInPool =
    info &&
    migrationAmount &&
    compareWithDecimalOffset(
      migrationAmount,
      info.fromDecimals,
      info.toPoolBalance,
      info.toDecimals
    ) !== 1;
  const migrateFromTokensInWallet =
    info &&
    migrationAmount &&
    compareWithDecimalOffset(
      migrationAmount,
      info.fromDecimals,
      info.fromWalletBalance,
      info.fromDecimals
    ) !== 1;
  const migrateIsReady = migrateFromTokensInWallet && migrateToTokensInPool;
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

  const redeemSharesInWallet =
    info &&
    redeemAmount &&
    compareWithDecimalOffset(
      redeemAmount,
      info.sharesDecimals,
      info.walletSharesBalance,
      info.sharesDecimals
    ) !== 1;
  const redeemFromTokensInPool =
    info &&
    redeemAmount &&
    compareWithDecimalOffset(
      redeemAmount,
      info.sharesDecimals,
      info.fromPoolBalance,
      info.fromDecimals
    ) !== 1;
  const redeemIsReady = redeemSharesInWallet && redeemFromTokensInPool;
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

  const topContent = (
    <>
      <Typography variant="h6">Manage an Ethereum Pool</Typography>
      <EthereumSignerKey />
      <TextField
        value={migratorAddress}
        onChange={(event) => setMigratorAddress(event.target.value)}
        label={"Migrator Address"}
        fullWidth
        style={{ display: "block" }}
      />
    </>
  );
  const infoDisplay = poolInfo.isLoading ? (
    <CircularProgress />
  ) : poolInfo.error ? (
    <Typography>{poolInfo.error}</Typography>
  ) : !poolInfo.data ? null : (
    <>
      <div style={{ display: "flex" }}>
        <div>
          <Typography variant="h5">Pool Balances</Typography>
          <Typography>
            {`'From' Asset: `}
            {info?.fromPoolBalance}
            <SmartAddress
              chainId={CHAIN_ID_ETH}
              address={info?.fromAddress}
              symbol={info?.fromSymbol}
            />
          </Typography>
          <Typography>
            {`'To' Asset: `}
            {info?.toPoolBalance}
            <SmartAddress
              chainId={CHAIN_ID_ETH}
              address={info?.toAddress}
              symbol={info?.toSymbol}
            />
          </Typography>
        </div>
        <div style={{ flexGrow: 1 }} />
        <div>
          <Typography variant="h5">Connected Wallet Balances</Typography>
          <Typography>
            {`'From' Asset: `}
            {info?.fromWalletBalance}
            <SmartAddress
              chainId={CHAIN_ID_ETH}
              address={info?.fromAddress}
              symbol={info?.fromSymbol}
            />
          </Typography>
          <Typography>
            {`'To' Asset: `}
            {info?.toWalletBalance}
            <SmartAddress
              chainId={CHAIN_ID_ETH}
              address={info?.toAddress}
              symbol={info?.toSymbol}
            />
          </Typography>
          <Typography>
            {`'Shares' Asset: `}
            {info?.walletSharesBalance}
            <SmartAddress chainId={CHAIN_ID_ETH} address={info?.poolAddress} />
          </Typography>
        </div>
      </div>
      <Button onClick={forceRefresh} variant="contained" color="primary">
        Force Refresh
      </Button>
    </>
  );

  const actionPanel = poolInfo.data ? (
    <>
      {addLiquidityUI}
      <Divider className={classes.divider} />
      {removeLiquidityUI}
      <Divider className={classes.divider} />
      {redeemSharesUI}
      <Divider className={classes.divider} />
      {migrateTokensUI}
    </>
  ) : null;

  return (
    <>
      <Container maxWidth="md" className={classes.rootContainer}>
        <Paper className={classes.mainPaper}>
          {topContent}
          {infoDisplay && (
            <>
              <Divider className={classes.divider} />
              {infoDisplay}
            </>
          )}
          {actionPanel && (
            <>
              <Divider className={classes.divider} />
              {actionPanel}
            </>
          )}
        </Paper>
        <LogWatcher />
      </Container>
    </>
  );
}

export default MigrateEthereum;
