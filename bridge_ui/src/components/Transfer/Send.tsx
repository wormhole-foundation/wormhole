import { Button, CircularProgress, makeStyles } from "@material-ui/core";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../../contexts/SolanaWalletContext";
import useWrappedAsset from "../../hooks/useWrappedAsset";
import {
  selectAmount,
  selectIsSendComplete,
  selectIsSending,
  selectIsTargetComplete,
  selectSourceAsset,
  selectSourceChain,
  selectSourceParsedTokenAccount,
  selectTargetChain,
} from "../../store/selectors";
import { setIsSending, setSignedVAAHex } from "../../store/transferSlice";
import { uint8ArrayToHex } from "../../utils/array";
import { CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "../../utils/consts";
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
  const isTargetComplete = useSelector(selectIsTargetComplete);
  const isSending = useSelector(selectIsSending);
  const isSendComplete = useSelector(selectIsSendComplete);
  const { provider, signer, signerAddress } = useEthereumProvider();
  const { wallet } = useSolanaWallet();
  const solPK = wallet?.publicKey;
  const sourceParsedTokenAccount = useSelector(selectSourceParsedTokenAccount);
  const tokenPK = sourceParsedTokenAccount?.publicKey;
  const decimals = sourceParsedTokenAccount?.decimals;
  const {
    isLoading: isCheckingWrapped,
    // isWrapped,
    wrappedAsset,
  } = useWrappedAsset(targetChain, sourceChain, sourceAsset, provider);
  // TODO: check this and send to separate flow
  const isWrapped = true;
  console.log(isCheckingWrapped, isWrapped, wrappedAsset);
  // TODO: dynamically get "to" wallet
  const handleTransferClick = useCallback(() => {
    // TODO: we should separate state for transaction vs fetching vaa
    // TODO: more generic way of calling these
    if (transferFrom[sourceChain]) {
      if (
        sourceChain === CHAIN_ID_ETH &&
        transferFrom[sourceChain] === transferFromEth &&
        decimals
      ) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          dispatch(setIsSending(true));
          try {
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
          } catch (e) {
            console.error(e);
            dispatch(setIsSending(false));
          }
        })();
      }
      if (
        sourceChain === CHAIN_ID_SOLANA &&
        transferFrom[sourceChain] === transferFromSolana &&
        decimals
      ) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          dispatch(setIsSending(true));
          try {
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
          } catch (e) {
            console.error(e);
            dispatch(setIsSending(false));
          }
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
  return (
    <>
      <div style={{ position: "relative" }}>
        <Button
          color="primary"
          variant="contained"
          className={classes.transferButton}
          onClick={handleTransferClick}
          disabled={!isTargetComplete || isSending || isSendComplete}
        >
          Transfer
        </Button>
        {isSending ? (
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
    </>
  );
}

export default Send;
