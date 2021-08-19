import { CHAIN_ID_TERRA, CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { Button, CircularProgress, makeStyles } from "@material-ui/core";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../../contexts/SolanaWalletContext";
import { useConnectedWallet } from "@terra-money/wallet-provider";
import { setIsSending, setSignedVAAHex } from "../../store/attestSlice";
import {
  selectAttestIsSendComplete,
  selectAttestIsSending,
  selectAttestIsTargetComplete,
  selectAttestSourceAsset,
  selectAttestSourceChain,
} from "../../store/selectors";
import { uint8ArrayToHex } from "../../utils/array";
import {
  attestFromEth,
  attestFromSolana,
  attestFromTerra,
} from "../../utils/attestFrom";

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
  const isTargetComplete = useSelector(selectAttestIsTargetComplete);
  const isSending = useSelector(selectAttestIsSending);
  const isSendComplete = useSelector(selectAttestIsSendComplete);
  const { signer } = useEthereumProvider();
  const { wallet } = useSolanaWallet();
  const terraWallet = useConnectedWallet();
  const solPK = wallet?.publicKey;
  // TODO: dynamically get "to" wallet
  const handleAttestClick = useCallback(() => {
    if (sourceChain === CHAIN_ID_ETH) {
      //TODO: just for testing, this should eventually use the store to communicate between steps
      (async () => {
        dispatch(setIsSending(true));
        try {
          const vaaBytes = await attestFromEth(signer, sourceAsset);
          console.log("bytes in attest", vaaBytes);
          vaaBytes && dispatch(setSignedVAAHex(uint8ArrayToHex(vaaBytes)));
        } catch (e) {
          console.error(e);
          dispatch(setIsSending(false));
        }
      })();
    } else if (sourceChain === CHAIN_ID_SOLANA) {
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
    } else if (sourceChain === CHAIN_ID_TERRA) {
        //TODO: just for testing, this should eventually use the store to communicate between steps
        (async () => {
          dispatch(setIsSending(true));
          try {
            const vaaBytes = await attestFromTerra(terraWallet, sourceAsset);
            console.log("bytes in attest", vaaBytes);
            vaaBytes && dispatch(setSignedVAAHex(uint8ArrayToHex(vaaBytes)));
          } catch (e) {
            console.error(e);
            dispatch(setIsSending(false));
          }
        })();
      }
  }, [dispatch, sourceChain, signer, wallet, solPK, sourceAsset]);
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
