import {
  Button,
  Grid,
  makeStyles,
  MenuItem,
  TextField,
  Typography,
} from "@material-ui/core";
import { useCallback, useState } from "react";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import useEthereumBalance from "../hooks/useEthereumBalance";
import {
  ChainId,
  CHAINS,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  ETH_TEST_TOKEN_ADDRESS,
} from "../utils/consts";
import transferFrom from "../utils/transferFrom";
import EthereumSignerKey from "./EthereumSignerKey";
import SolanaWalletKey from "./SolanaWalletKey";

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
  const ethBalance = useEthereumBalance(assetAddress, provider);
  const { wallet } = useSolanaWallet();
  const solPK = wallet?.publicKey?.toBytes();
  // TODO: dynamically get "to" wallet
  const handleClick = useCallback(() => {
    if (transferFrom[fromChain]) {
      transferFrom[fromChain](provider, assetAddress, amount, toChain, solPK);
    }
  }, [fromChain, provider, solPK, assetAddress, amount, toChain]);
  // update this as we develop, just setting expectations with the button state
  const isTransferImplemented = !!transferFrom[fromChain];
  const isProviderConnected = !!provider;
  const isRecipientAvailable = !!solPK;
  const isAddressDefined = !!assetAddress;
  const isAmountPositive = Number(amount) > 0; // TODO: this needs per-chain, bn parsing
  const isBalanceAtLeastAmount = Number(ethBalance) >= Number(amount); // TODO: ditto
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
          <Typography>To</Typography>
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
          <EthereumSignerKey />
          <Typography>{ethBalance}</Typography>
        </Grid>
        <Grid item xs={4} className={classes.arrow}>
          &rarr;
        </Grid>
        <Grid item xs={4}>
          <Typography>From</Typography>
          <TextField select fullWidth value={toChain} onChange={handleToChange}>
            {CHAINS.map(({ id, name }) => (
              <MenuItem key={id} value={id}>
                {name}
              </MenuItem>
            ))}
          </TextField>
          <SolanaWalletKey />
        </Grid>
      </Grid>
      <TextField
        placeholder="Asset"
        fullWidth
        className={classes.transferField}
        value={assetAddress}
        onChange={handleAssetChange}
      />
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
        onClick={handleClick}
        disabled={!canAttemptTransfer}
      >
        Transfer
      </Button>
      {canAttemptTransfer ? null : (
        <Typography variant="body2" color="error">
          {!isTransferImplemented
            ? `Transfer is not yet implemented for ${CHAINS[fromChain]}`
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
            : !isBalanceAtLeastAmount
            ? "The amount may not be greater than the balance"
            : ""}
        </Typography>
      )}
    </div>
  );
}

export default Transfer;
