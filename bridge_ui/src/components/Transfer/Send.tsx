import {
  Button,
  CircularProgress,
  makeStyles,
  Typography,
} from "@material-ui/core";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../../contexts/SolanaWalletContext";
import useWrappedAsset from "../../hooks/useWrappedAsset";
import {
  selectAmount,
  selectSourceAsset,
  selectSourceChain,
  selectSourceParsedTokenAccount,
  selectTargetChain,
} from "../../store/selectors";
import { setSignedVAAHex } from "../../store/transferSlice";
import { uint8ArrayToHex } from "../../utils/array";
import attestFrom, {
  attestFromEth,
  attestFromSolana,
} from "../../utils/attestFrom";
import {
  CHAINS_BY_ID,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "../../utils/consts";
import transferFrom, {
  transferFromEth,
  transferFromSolana,
} from "../../utils/transferFrom";

const useStyles = makeStyles((theme) => ({
  transferButton: {
    marginTop: theme.spacing(2),
    textTransform: "none",
    width: "100%",
  },
}));

// TODO: move attest to its own workflow

function Send() {
  const classes = useStyles();
  const dispatch = useDispatch();
  const sourceChain = useSelector(selectSourceChain);
  const sourceAsset = useSelector(selectSourceAsset);
  const amount = useSelector(selectAmount);
  const targetChain = useSelector(selectTargetChain);
  const { provider, signer, signerAddress } = useEthereumProvider();
  const { wallet } = useSolanaWallet();
  const solPK = wallet?.publicKey;
  const sourceParsedTokenAccount = useSelector(selectSourceParsedTokenAccount);
  const tokenPK = sourceParsedTokenAccount?.publicKey;
  const decimals = sourceParsedTokenAccount?.decimals;
  const uiAmountString = sourceParsedTokenAccount?.uiAmountString;
  const {
    isLoading: isCheckingWrapped,
    // isWrapped,
    wrappedAsset,
  } = useWrappedAsset(targetChain, sourceChain, sourceAsset, provider);
  // TODO: check this and send to separate flow
  const isWrapped = true;
  console.log(isCheckingWrapped, isWrapped, wrappedAsset);
  const handleAttestClick = useCallback(() => {
    // TODO: more generic way of calling these
    if (attestFrom[sourceChain]) {
      if (
        sourceChain === CHAIN_ID_ETH &&
        attestFrom[sourceChain] === attestFromEth
      ) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          const vaaBytes = await attestFromEth(provider, signer, sourceAsset);
          console.log("bytes in transfer", vaaBytes);
        })();
      }
      if (
        sourceChain === CHAIN_ID_SOLANA &&
        attestFrom[sourceChain] === attestFromSolana &&
        decimals
      ) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          const vaaBytes = await attestFromSolana(
            wallet,
            solPK?.toString(),
            sourceAsset,
            decimals
          );
          console.log("bytes in transfer", vaaBytes);
        })();
      }
    }
  }, [sourceChain, provider, signer, wallet, solPK, sourceAsset, decimals]);
  // TODO: dynamically get "to" wallet
  const handleTransferClick = useCallback(() => {
    // TODO: more generic way of calling these
    if (transferFrom[sourceChain]) {
      if (
        sourceChain === CHAIN_ID_ETH &&
        transferFrom[sourceChain] === transferFromEth &&
        decimals
      ) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          const vaaBytes = await transferFromEth(
            provider,
            signer,
            sourceAsset,
            decimals,
            amount,
            targetChain,
            solPK?.toBytes()
          );
          console.log("bytes in transfer", vaaBytes);
          vaaBytes && dispatch(setSignedVAAHex(uint8ArrayToHex(vaaBytes)));
        })();
      }
      if (
        sourceChain === CHAIN_ID_SOLANA &&
        transferFrom[sourceChain] === transferFromSolana &&
        decimals
      ) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          const vaaBytes = await transferFromSolana(
            wallet,
            solPK?.toString(),
            tokenPK,
            sourceAsset,
            amount,
            decimals,
            signerAddress,
            targetChain
          );
          console.log("bytes in transfer", vaaBytes);
          vaaBytes && dispatch(setSignedVAAHex(uint8ArrayToHex(vaaBytes)));
        })();
      }
    }
  }, [
    dispatch,
    sourceChain,
    provider,
    signer,
    signerAddress,
    wallet,
    solPK,
    tokenPK,
    sourceAsset,
    amount,
    decimals,
    targetChain,
  ]);
  // update this as we develop, just setting expectations with the button state
  const hasDecimals = decimals !== undefined;
  const balance = Number(uiAmountString);
  const isAttestImplemented = !!attestFrom[sourceChain];
  const isTransferImplemented = !!transferFrom[sourceChain];
  const isProviderConnected = !!provider;
  const isRecipientAvailable = !!solPK;
  const isAddressDefined = !!sourceAsset;
  const isAmountPositive = Number(amount) > 0; // TODO: this needs per-chain, bn parsing
  const isBalanceAtLeastAmount = balance >= Number(amount); // TODO: ditto
  const canAttemptAttest =
    hasDecimals &&
    isAttestImplemented &&
    isProviderConnected &&
    isRecipientAvailable &&
    isAddressDefined;
  const canAttemptTransfer =
    hasDecimals &&
    isTransferImplemented &&
    isProviderConnected &&
    isRecipientAvailable &&
    isAddressDefined &&
    isAmountPositive &&
    isBalanceAtLeastAmount;
  return isWrapped ? (
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
            ? `Transfer is not yet implemented for ${CHAINS_BY_ID[sourceChain].name}`
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
          This token does not exist on {CHAINS_BY_ID[targetChain].name}. Someone
          must attest the the token to the target chain before it can be
          transferred.
        </Typography>
      ) : (
        <Typography variant="body2" color="error">
          {!isAttestImplemented
            ? `Transfer is not yet implemented for ${CHAINS_BY_ID[sourceChain].name}`
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
  );
}

export default Send;
