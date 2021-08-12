import { Button, CircularProgress, makeStyles } from "@material-ui/core";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { PublicKey } from "@solana/web3.js";
import { zeroPad } from "ethers/lib/utils";
import { useCallback, useEffect, useRef } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../../contexts/SolanaWalletContext";
import {
  selectTransferAmount,
  selectTransferIsSendComplete,
  selectTransferIsSending,
  selectTransferIsTargetComplete,
  selectTransferOriginAsset,
  selectTransferOriginChain,
  selectTransferSourceAsset,
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccount,
  selectTransferTargetAsset,
  selectTransferTargetChain,
  selectTransferTargetParsedTokenAccount,
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
  const sourceChain = useSelector(selectTransferSourceChain);
  const sourceAsset = useSelector(selectTransferSourceAsset);
  const originChain = useSelector(selectTransferOriginChain);
  const originAsset = useSelector(selectTransferOriginAsset);
  const amount = useSelector(selectTransferAmount);
  const targetChain = useSelector(selectTransferTargetChain);
  const targetAsset = useSelector(selectTransferTargetAsset);
  const isTargetComplete = useSelector(selectTransferIsTargetComplete);
  const isSending = useSelector(selectTransferIsSending);
  const isSendComplete = useSelector(selectTransferIsSendComplete);
  const { provider, signer, signerAddress } = useEthereumProvider();
  const { wallet } = useSolanaWallet();
  const solPK = wallet?.publicKey;
  const sourceParsedTokenAccount = useSelector(
    selectTransferSourceParsedTokenAccount
  );
  const sourceTokenPublicKey = sourceParsedTokenAccount?.publicKey;
  const decimals = sourceParsedTokenAccount?.decimals;
  const targetParsedTokenAccount = useSelector(
    selectTransferTargetParsedTokenAccount
  );
  // TODO: we probably shouldn't get here if we don't have this public key
  // TODO: also this is just for solana... send help(ers)
  const targetTokenAccountPublicKey = targetParsedTokenAccount?.publicKey;
  console.log(
    "Sending to:",
    targetTokenAccountPublicKey,
    targetTokenAccountPublicKey &&
      new PublicKey(targetTokenAccountPublicKey).toBytes()
  );
  // TODO: AVOID THIS DANGEROUS CACOPHONY
  const tpkRef = useRef<undefined | Uint8Array>(undefined);
  useEffect(() => {
    (async () => {
      if (targetChain === CHAIN_ID_SOLANA) {
        tpkRef.current = targetTokenAccountPublicKey
          ? zeroPad(new PublicKey(targetTokenAccountPublicKey).toBytes(), 32) // use the target's TokenAccount if it exists
          : solPK && targetAsset // otherwise, use the associated token account (which we create in the case it doesn't exist)
          ? zeroPad(
              (
                await Token.getAssociatedTokenAddress(
                  ASSOCIATED_TOKEN_PROGRAM_ID,
                  TOKEN_PROGRAM_ID,
                  new PublicKey(targetAsset),
                  solPK
                )
              ).toBytes(),
              32
            )
          : undefined;
      } else tpkRef.current = undefined;
    })();
  }, [targetChain, solPK, targetAsset, targetTokenAccountPublicKey]);
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
            console.log("actually sending", tpkRef.current);
            const vaaBytes = await transferFromEth(
              provider,
              signer,
              sourceAsset,
              decimals,
              amount,
              targetChain,
              tpkRef.current
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
              sourceTokenPublicKey,
              sourceAsset,
              amount, //TODO: avoid decimals, pass in parsed amount
              decimals,
              signerAddress,
              targetChain,
              originAsset,
              originChain
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
    sourceTokenPublicKey,
    sourceAsset,
    amount,
    decimals,
    targetChain,
    originAsset,
    originChain,
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
