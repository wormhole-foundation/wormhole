import { Button, CircularProgress, makeStyles } from "@material-ui/core";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../../contexts/SolanaWalletContext";
import useWrappedAsset from "../../hooks/useWrappedAsset";
import { setIsSending, setSignedVAAHex } from "../../store/attestSlice";
import {
  selectAttestIsSendComplete,
  selectAttestIsSending,
  selectAttestIsTargetComplete,
  selectAttestSourceAsset,
  selectAttestSourceChain,
  selectAttestTargetChain,
} from "../../store/selectors";
import { uint8ArrayToHex } from "../../utils/array";
import attestFrom, {
  attestFromEth,
  attestFromSolana,
} from "../../utils/attestFrom";
import { CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "../../utils/consts";

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
  const sourceChain = useSelector(selectAttestSourceChain);
  const sourceAsset = useSelector(selectAttestSourceAsset);
  const targetChain = useSelector(selectAttestTargetChain);
  const isTargetComplete = useSelector(selectAttestIsTargetComplete);
  const isSending = useSelector(selectAttestIsSending);
  const isSendComplete = useSelector(selectAttestIsSendComplete);
  const { provider, signer } = useEthereumProvider();
  const { wallet } = useSolanaWallet();
  const solPK = wallet?.publicKey;
  const {
    isLoading: isCheckingWrapped,
    // isWrapped,
    wrappedAsset,
  } = useWrappedAsset(targetChain, sourceChain, sourceAsset, provider);
  // TODO: check this and send to separate flow
  const isWrapped = true;
  console.log(isCheckingWrapped, isWrapped, wrappedAsset);
  // TODO: dynamically get "to" wallet
  const handleAttestClick = useCallback(() => {
    // TODO: more generic way of calling these
    if (attestFrom[sourceChain]) {
      if (
        sourceChain === CHAIN_ID_ETH &&
        attestFrom[sourceChain] === attestFromEth
      ) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          dispatch(setIsSending(true));
          try {
            const vaaBytes = await attestFromEth(provider, signer, sourceAsset);
            console.log("bytes in attest", vaaBytes);
            vaaBytes && dispatch(setSignedVAAHex(uint8ArrayToHex(vaaBytes)));
          } catch (e) {
            console.error(e);
            dispatch(setIsSending(false));
          }
        })();
      }
      if (
        sourceChain === CHAIN_ID_SOLANA &&
        attestFrom[sourceChain] === attestFromSolana
      ) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          dispatch(setIsSending(true));
          try {
            const vaaBytes = await attestFromSolana(
              wallet,
              solPK?.toString(),
              sourceAsset
            );
            console.log("bytes in attest", vaaBytes);
            vaaBytes && dispatch(setSignedVAAHex(uint8ArrayToHex(vaaBytes)));
          } catch (e) {
            console.error(e);
            dispatch(setIsSending(false));
          }
        })();
      }
    }
  }, [dispatch, sourceChain, provider, signer, wallet, solPK, sourceAsset]);
  return (
    <>
      <div style={{ position: "relative" }}>
        <Button
          color="primary"
          variant="contained"
          className={classes.transferButton}
          onClick={handleAttestClick}
          disabled={!isTargetComplete || isSending || isSendComplete}
        >
          Attest
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
