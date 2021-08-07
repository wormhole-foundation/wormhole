import {
  Button,
  CircularProgress,
  Container,
  makeStyles,
  MenuItem,
  Step,
  StepButton,
  StepContent,
  Stepper,
  TextField,
  Typography,
} from "@material-ui/core";
import { useCallback, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import useEthereumBalance from "../hooks/useEthereumBalance";
import useSolanaBalance from "../hooks/useSolanaBalance";
import useWrappedAsset from "../hooks/useWrappedAsset";
import {
  selectActiveStep,
  selectSignedVAA,
  selectSourceChain,
  selectTargetChain,
} from "../store/selectors";
import {
  incrementStep,
  setSignedVAA,
  setSourceChain,
  setStep,
  setTargetChain,
} from "../store/transferSlice";
import attestFrom, {
  attestFromEth,
  attestFromSolana,
} from "../utils/attestFrom";
import {
  CHAINS,
  CHAINS_BY_ID,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  ETH_TEST_TOKEN_ADDRESS,
  SOL_TEST_TOKEN_ADDRESS,
} from "../utils/consts";
import redeemOn, { redeemOnEth } from "../utils/redeemOn";
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
    marginTop: theme.spacing(2),
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
  const dispatch = useDispatch();
  const activeStep = useSelector(selectActiveStep);
  const fromChain = useSelector(selectSourceChain);
  const toChain = useSelector(selectTargetChain);
  const [assetAddress, setAssetAddress] = useState(SOL_TEST_TOKEN_ADDRESS);
  const [amount, setAmount] = useState("");
  const handleFromChange = useCallback(
    (event) => {
      dispatch(setSourceChain(event.target.value));
      // TODO: remove or check env - for testing purposes
      if (event.target.value === CHAIN_ID_ETH) {
        setAssetAddress(ETH_TEST_TOKEN_ADDRESS);
      }
      if (event.target.value === CHAIN_ID_SOLANA) {
        setAssetAddress(SOL_TEST_TOKEN_ADDRESS);
      }
      if (toChain === event.target.value) {
        dispatch(setTargetChain(fromChain));
      }
    },
    [dispatch, fromChain, toChain]
  );
  const handleToChange = useCallback(
    (event) => {
      dispatch(setTargetChain(event.target.value));
      if (fromChain === event.target.value) {
        dispatch(setSourceChain(toChain));
        // TODO: remove or check env - for testing purposes
        if (toChain === CHAIN_ID_ETH) {
          setAssetAddress(ETH_TEST_TOKEN_ADDRESS);
        }
        if (toChain === CHAIN_ID_SOLANA) {
          setAssetAddress(SOL_TEST_TOKEN_ADDRESS);
        }
      }
    },
    [dispatch, fromChain, toChain]
  );
  const handleAssetChange = useCallback((event) => {
    setAssetAddress(event.target.value);
  }, []);
  const handleAmountChange = useCallback((event) => {
    setAmount(event.target.value);
  }, []);
  const { provider, signer, signerAddress } = useEthereumProvider();
  const { decimals: ethDecimals, uiAmountString: ethBalance } =
    useEthereumBalance(
      assetAddress,
      signerAddress,
      provider,
      fromChain === CHAIN_ID_ETH
    );
  const { wallet } = useSolanaWallet();
  const solPK = wallet?.publicKey;
  const {
    tokenAccount: solTokenPK,
    decimals: solDecimals,
    uiAmount: solBalance,
  } = useSolanaBalance(assetAddress, solPK, fromChain === CHAIN_ID_SOLANA);
  const {
    isLoading: isCheckingWrapped,
    // isWrapped,
    wrappedAsset,
  } = useWrappedAsset(toChain, fromChain, assetAddress, provider);
  const isWrapped = true;
  console.log(isCheckingWrapped, isWrapped, wrappedAsset);
  const handleAttestClick = useCallback(() => {
    // TODO: more generic way of calling these
    if (attestFrom[fromChain]) {
      if (
        fromChain === CHAIN_ID_ETH &&
        attestFrom[fromChain] === attestFromEth
      ) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          const vaaBytes = await attestFromEth(provider, signer, assetAddress);
          console.log("bytes in transfer", vaaBytes);
        })();
      }
      if (
        fromChain === CHAIN_ID_SOLANA &&
        attestFrom[fromChain] === attestFromSolana
      ) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          const vaaBytes = await attestFromSolana(
            wallet,
            solPK?.toString(),
            assetAddress,
            solDecimals
          );
          console.log("bytes in transfer", vaaBytes);
        })();
      }
    }
  }, [fromChain, provider, signer, wallet, solPK, assetAddress, solDecimals]);
  // TODO: dynamically get "to" wallet
  const handleTransferClick = useCallback(() => {
    // TODO: more generic way of calling these
    if (transferFrom[fromChain]) {
      if (
        fromChain === CHAIN_ID_ETH &&
        transferFrom[fromChain] === transferFromEth
      ) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          const vaaBytes = await transferFromEth(
            provider,
            signer,
            assetAddress,
            ethDecimals,
            amount,
            toChain,
            solPK?.toBytes()
          );
          console.log("bytes in transfer", vaaBytes);
          vaaBytes && dispatch(setSignedVAA(vaaBytes));
        })();
      }
      if (
        fromChain === CHAIN_ID_SOLANA &&
        transferFrom[fromChain] === transferFromSolana
      ) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          const vaaBytes = await transferFromSolana(
            wallet,
            solPK?.toString(),
            solTokenPK?.toString(),
            assetAddress,
            amount,
            solDecimals,
            signerAddress,
            toChain
          );
          console.log("bytes in transfer", vaaBytes);
          vaaBytes && dispatch(setSignedVAA(vaaBytes));
        })();
      }
    }
  }, [
    dispatch,
    fromChain,
    provider,
    signer,
    signerAddress,
    wallet,
    solPK,
    solTokenPK,
    assetAddress,
    amount,
    ethDecimals,
    solDecimals,
    toChain,
  ]);
  const signedVAA = useSelector(selectSignedVAA);
  const handleRedeemClick = useCallback(() => {
    if (
      toChain === CHAIN_ID_ETH &&
      redeemOn[toChain] === redeemOnEth &&
      signedVAA
    ) {
      redeemOnEth(provider, signer, signedVAA);
    }
  }, [toChain, provider, signer, signedVAA]);
  // update this as we develop, just setting expectations with the button state
  const balance = Number(ethBalance) || solBalance;
  const isAttestImplemented = !!attestFrom[fromChain];
  const isTransferImplemented = !!transferFrom[fromChain];
  const isProviderConnected = !!provider;
  const isRecipientAvailable = !!solPK;
  const isAddressDefined = !!assetAddress;
  const isAmountPositive = Number(amount) > 0; // TODO: this needs per-chain, bn parsing
  const isBalanceAtLeastAmount = balance >= Number(amount); // TODO: ditto
  const canAttemptAttest =
    isAttestImplemented &&
    isProviderConnected &&
    isRecipientAvailable &&
    isAddressDefined;
  const canAttemptTransfer =
    isTransferImplemented &&
    isProviderConnected &&
    isRecipientAvailable &&
    isAddressDefined &&
    isAmountPositive &&
    isBalanceAtLeastAmount;
  const handleNextClick = useCallback(() => {
    dispatch(incrementStep());
  }, [dispatch]);
  return (
    <Container maxWidth="md">
      <Stepper activeStep={activeStep} orientation="vertical">
        <Step>
          <StepButton onClick={() => dispatch(setStep(0))}>
            Select a source
          </StepButton>
          <StepContent>
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
              onClick={handleNextClick}
              variant="contained"
              color="primary"
            >
              Next
            </Button>
          </StepContent>
        </Step>
        <Step>
          <StepButton onClick={() => dispatch(setStep(1))}>
            Select a target
          </StepButton>
          <StepContent>
            <TextField
              select
              fullWidth
              value={toChain}
              onChange={handleToChange}
            >
              {CHAINS.map(({ id, name }) => (
                <MenuItem key={id} value={id}>
                  {name}
                </MenuItem>
              ))}
            </TextField>
            {/* TODO: determine "to" token address */}
            <KeyAndBalance chainId={toChain} />
            <Button
              onClick={handleNextClick}
              variant="contained"
              color="primary"
            >
              Next
            </Button>
          </StepContent>
        </Step>
        <Step>
          <StepButton onClick={() => dispatch(setStep(2))}>
            Send tokens
          </StepButton>
          <StepContent>
            {isWrapped ? (
              <>
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
                    disabled={isCheckingWrapped || !canAttemptAttest}
                    onClick={handleAttestClick}
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
                {isCheckingWrapped ? null : canAttemptAttest ? (
                  <Typography variant="body2">
                    <br />
                    This token does not exist on {CHAINS_BY_ID[toChain].name}.
                    Someone must attest the the token to the target chain before
                    it can be transferred.
                  </Typography>
                ) : (
                  <Typography variant="body2" color="error">
                    {!isAttestImplemented
                      ? `Transfer is not yet implemented for ${CHAINS_BY_ID[fromChain].name}`
                      : !isProviderConnected
                      ? "The source wallet is not connected"
                      : !isRecipientAvailable
                      ? "The receiving wallet is not connected"
                      : !isAddressDefined
                      ? "Please provide an asset address"
                      : ""}
                  </Typography>
                )}
              </>
            )}
          </StepContent>
        </Step>
        <Step>
          <StepButton
            onClick={() => dispatch(setStep(3))}
            disabled={!signedVAA}
          >
            Redeem tokens
          </StepButton>
          <StepContent>
            <Button
              color="primary"
              variant="contained"
              className={classes.transferButton}
              onClick={handleRedeemClick}
            >
              Redeem
            </Button>
          </StepContent>
        </Step>
      </Stepper>
    </Container>
  );
}

export default Transfer;
