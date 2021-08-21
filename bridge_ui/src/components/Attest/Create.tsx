import {
  CHAIN_ID_TERRA,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import { Button, CircularProgress, makeStyles } from "@material-ui/core";
import { useCallback } from "react";
import { useConnectedWallet } from "@terra-money/wallet-provider";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../../contexts/SolanaWalletContext";
import useAttestSignedVAA from "../../hooks/useAttestSignedVAA";
import { reset, setIsCreating } from "../../store/attestSlice";
import {
  selectAttestIsCreating,
  selectAttestTargetChain,
} from "../../store/selectors";
import {
  createWrappedOnEth,
  createWrappedOnSolana,
  createWrappedOnTerra,
} from "../../utils/createWrappedOn";

const useStyles = makeStyles((theme) => ({
  transferButton: {
    marginTop: theme.spacing(2),
    textTransform: "none",
    width: "100%",
  },
}));

function Create() {
  const dispatch = useDispatch();
  const classes = useStyles();
  const targetChain = useSelector(selectAttestTargetChain);
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const signedVAA = useAttestSignedVAA();
  const isCreating = useSelector(selectAttestIsCreating);
  const { signer } = useEthereumProvider();
  const terraWallet = useConnectedWallet();
  const handleCreateClick = useCallback(() => {
    if (targetChain === CHAIN_ID_SOLANA && signedVAA) {
      (async () => {
        dispatch(setIsCreating(true));
        await createWrappedOnSolana(solanaWallet, solPK?.toString(), signedVAA);
        dispatch(reset());
      })();
    }
    if (targetChain === CHAIN_ID_ETH && signedVAA) {
      (async () => {
        dispatch(setIsCreating(true));
        await createWrappedOnEth(signer, signedVAA);
        dispatch(reset());
      })();
    }
    if (targetChain === CHAIN_ID_TERRA && signedVAA) {
      (async () => {
        dispatch(setIsCreating(true));
        createWrappedOnTerra(terraWallet, signedVAA);
      })();
    }
  }, [dispatch, targetChain, solanaWallet, solPK, signedVAA, signer]);
  return (
    <div style={{ position: "relative" }}>
      <Button
        color="primary"
        variant="contained"
        className={classes.transferButton}
        disabled={isCreating}
        onClick={handleCreateClick}
      >
        Create
      </Button>
      {isCreating ? (
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
  );
}

export default Create;
