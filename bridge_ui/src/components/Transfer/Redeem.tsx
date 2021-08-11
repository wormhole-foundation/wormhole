import { Button, CircularProgress, makeStyles } from "@material-ui/core";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import useTransferSignedVAA from "../../hooks/useTransferSignedVAA";
import {
  selectTransferIsRedeeming,
  selectTransferTargetChain,
} from "../../store/selectors";
import { setIsRedeeming } from "../../store/transferSlice";
import { CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "../../utils/consts";
import redeemOn, { redeemOnEth, redeemOnSolana } from "../../utils/redeemOn";
import { useSolanaWallet } from "../../contexts/SolanaWalletContext";

const useStyles = makeStyles((theme) => ({
  transferButton: {
    marginTop: theme.spacing(2),
    textTransform: "none",
    width: "100%",
  },
}));

function Redeem() {
  const dispatch = useDispatch();
  const classes = useStyles();
  const targetChain = useSelector(selectTransferTargetChain);
  const { wallet } = useSolanaWallet();
  const solPK = wallet?.publicKey;
  const { provider, signer } = useEthereumProvider();
  const signedVAA = useTransferSignedVAA();
  const isRedeeming = useSelector(selectTransferIsRedeeming);
  const handleRedeemClick = useCallback(() => {
    if (
      targetChain === CHAIN_ID_ETH &&
      redeemOn[targetChain] === redeemOnEth &&
      signedVAA
    ) {
      dispatch(setIsRedeeming(true));
      redeemOnEth(provider, signer, signedVAA);
    }
    if (
      targetChain === CHAIN_ID_SOLANA &&
      redeemOn[targetChain] === redeemOnSolana &&
      signedVAA
    ) {
      dispatch(setIsRedeeming(true));
      redeemOnSolana(wallet, solPK?.toString(), signedVAA);
    }
  }, [dispatch, targetChain, provider, signer, signedVAA, wallet, solPK]);
  return (
    <div style={{ position: "relative" }}>
      <Button
        color="primary"
        variant="contained"
        className={classes.transferButton}
        disabled={isRedeeming}
        onClick={handleRedeemClick}
      >
        Redeem
      </Button>
      {isRedeeming ? (
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

export default Redeem;
