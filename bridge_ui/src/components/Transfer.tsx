import {
  Button,
  CircularProgress,
  Grid,
  makeStyles,
  MenuItem,
  TextField,
  Typography,
} from "@material-ui/core";
import { ethers } from "ethers";
import { useCallback, useState } from "react";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import useEthereumBalance from "../hooks/useEthereumBalance";
import useSolanaBalance from "../hooks/useSolanaBalance";
import useWrappedAsset from "../hooks/useWrappedAsset";
import {
  ChainId,
  CHAINS,
  CHAINS_BY_ID,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  ETH_TEST_TOKEN_ADDRESS,
  SOL_TEST_TOKEN_ADDRESS,
} from "../utils/consts";
import transferFrom, {
  transferFromEth,
  transferFromSolana,
} from "../utils/transferFrom";
import KeyAndBalance from "./KeyAndBalance";

const useStyles = makeStyles((theme) => ({
  transferBox: {
    width: 540,
    margin: "auto",
    border: `.5px solid ${theme.palette.divider}`,
    padding: theme.spacing(5.5, 12),
  },
  arrow: {
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
  },
  transferField: {
    marginTop: theme.spacing(5),
  },
  transferButton: {
    marginTop: theme.spacing(7.5),
    textTransform: "none",
    width: "100%",
  },
}));

// TODO: ensure that both wallets are connected to the same known network
// TODO: loaders and such, navigation block?
// TODO: refresh displayed token amount after transfer somehow, could be resolved by having different components appear
// TODO: warn if amount exceeds balance

function Transfer() {
  const classes = useStyles();
  //TODO: don't attempt to connect to any wallets until the user clicks a connect button
  const [fromChain, setFromChain] = useState<ChainId>(CHAIN_ID_ETH);
  const [toChain, setToChain] = useState<ChainId>(CHAIN_ID_SOLANA);
  const [assetAddress, setAssetAddress] = useState(ETH_TEST_TOKEN_ADDRESS);
  const [amount, setAmount] = useState("");
  const handleFromChange = useCallback(
    (event) => {
      setFromChain(event.target.value);
      // TODO: remove or check env - for testing purposes
      if (event.target.value === CHAIN_ID_ETH) {
        setAssetAddress(ETH_TEST_TOKEN_ADDRESS);
      }
      if (event.target.value === CHAIN_ID_SOLANA) {
        setAssetAddress(SOL_TEST_TOKEN_ADDRESS);
      }
      if (toChain === event.target.value) {
        setToChain(fromChain);
      }
    },
    [fromChain, toChain]
  );
  const handleToChange = useCallback(
    (event) => {
      setToChain(event.target.value);
      if (fromChain === event.target.value) {
        setFromChain(toChain);
        // TODO: remove or check env - for testing purposes
        if (toChain === CHAIN_ID_ETH) {
          setAssetAddress(ETH_TEST_TOKEN_ADDRESS);
        }
        if (toChain === CHAIN_ID_SOLANA) {
          setAssetAddress(SOL_TEST_TOKEN_ADDRESS);
        }
      }
    },
    [fromChain, toChain]
  );
  const handleAssetChange = useCallback((event) => {
    setAssetAddress(event.target.value);
  }, []);
  const handleAmountChange = useCallback((event) => {
    setAmount(event.target.value);
  }, []);
  const provider = useEthereumProvider();
  const { wallet } = useSolanaWallet();
  const solPK = wallet?.publicKey;
  const {
    tokenAccount: solTokenPK,
    decimals: solDecimals,
    uiAmount: solBalance,
  } = useSolanaBalance(assetAddress, solPK, fromChain === CHAIN_ID_SOLANA);
  const { isLoading: isCheckingWrapped, wrappedAsset } = useWrappedAsset(
    toChain,
    fromChain,
    assetAddress,
    provider
  );
  console.log(isCheckingWrapped, wrappedAsset);
  // TODO: make a helper function for this
  const isWrapped = true;
  // wrappedAsset && wrappedAsset !== ethers.constants.AddressZero;
  // TODO: dynamically get "to" wallet
  const handleTransferClick = useCallback(() => {
    // TODO: more generic way of calling these
    if (transferFrom[fromChain]) {
      if (
        fromChain === CHAIN_ID_ETH &&
        transferFrom[fromChain] === transferFromEth
      ) {
        transferFromEth(
          provider,
          assetAddress,
          amount,
          toChain,
          solPK?.toBytes()
        );
      }
      if (
        fromChain === CHAIN_ID_SOLANA &&
        transferFrom[fromChain] === transferFromSolana
      ) {
        transferFromSolana(
          wallet,
          solPK?.toString(),
          solTokenPK?.toString(),
          assetAddress,
          amount,
          solDecimals,
          provider,
          toChain
        );
      }
    }
  }, [
    fromChain,
    provider,
    wallet,
    solPK,
    solTokenPK,
    assetAddress,
    amount,
    solDecimals,
    toChain,
  ]);
  // update this as we develop, just setting expectations with the button state
  const ethBalance = useEthereumBalance(
    assetAddress,
    provider,
    fromChain === CHAIN_ID_ETH
  );
  const balance = Number(ethBalance) || solBalance;
  const isTransferImplemented = !!transferFrom[fromChain];
  const isProviderConnected = !!provider;
  const isRecipientAvailable = !!solPK;
  const isAddressDefined = !!assetAddress;
  const isAmountPositive = Number(amount) > 0; // TODO: this needs per-chain, bn parsing
  const isBalanceAtLeastAmount = balance >= Number(amount); // TODO: ditto
  const canAttemptTransfer =
    isTransferImplemented &&
    isProviderConnected &&
    isRecipientAvailable &&
    isAddressDefined &&
    isAmountPositive &&
    isBalanceAtLeastAmount;
  return (
    <div className={classes.transferBox}>
      <Grid container>
        <Grid item xs={4}>
          <Typography>From</Typography>
          <TextField
            select
            fullWidth
            value={fromChain}
            onChange={handleFromChange}
          >
            {CHAINS.map(({ id, name }) => (
              <MenuItem key={id} value={id}>
                {name}
              </MenuItem>
            ))}
          </TextField>
          <KeyAndBalance chainId={fromChain} tokenAddress={assetAddress} />
        </Grid>
        <Grid item xs={4} className={classes.arrow}>
          &rarr;
        </Grid>
        <Grid item xs={4}>
          <Typography>To</Typography>
          <TextField select fullWidth value={toChain} onChange={handleToChange}>
            {CHAINS.map(({ id, name }) => (
              <MenuItem key={id} value={id}>
                {name}
              </MenuItem>
            ))}
          </TextField>
          {/* TODO: determine "to" token address */}
          <KeyAndBalance chainId={toChain} />
        </Grid>
      </Grid>
      <TextField
        placeholder="Asset"
        fullWidth
        className={classes.transferField}
        value={assetAddress}
        onChange={handleAssetChange}
      />
      {isWrapped ? (
        <>
          <TextField
            placeholder="Amount"
            type="number"
            fullWidth
            className={classes.transferField}
            value={amount}
            onChange={handleAmountChange}
          />
          <Button
            color="primary"
            variant="contained"
            className={classes.transferButton}
            onClick={handleTransferClick}
            disabled={!canAttemptTransfer}
          >
            Transfer
          </Button>
          {canAttemptTransfer ? null : (
            <Typography variant="body2" color="error">
              {!isTransferImplemented
                ? `Transfer is not yet implemented for ${CHAINS_BY_ID[fromChain].name}`
                : !isProviderConnected
                ? "The source wallet is not connected"
                : !isRecipientAvailable
                ? "The receiving wallet is not connected"
                : !isAddressDefined
                ? "Please provide an asset address"
                : !isAmountPositive
                ? "The amount must be positive"
                : !isBalanceAtLeastAmount
                ? "The amount may not be greater than the balance"
                : ""}
            </Typography>
          )}
        </>
      ) : (
        <>
          <div style={{ position: "relative" }}>
            <Button
              color="primary"
              variant="contained"
              disabled={isCheckingWrapped}
              className={classes.transferButton}
            >
              Attest
            </Button>
            {isCheckingWrapped ? (
              <CircularProgress
                size={24}
                color="inherit"
                style={{
                  position: "absolute",
                  bottom: 0,
                  left: "50%",
                  marginLeft: -12,
                  marginBottom: 6,
                }}
              />
            ) : null}
          </div>
          {isCheckingWrapped ? null : (
            <Typography variant="body2">
              <br />
              This token does not exist on {CHAINS_BY_ID[toChain].name}. Someone
              must attest the the token to the target chain before it can be
              transferred.
            </Typography>
          )}
        </>
      )}
    </div>
  );
}

export default Transfer;
