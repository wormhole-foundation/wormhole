import { Button, CircularProgress, makeStyles } from "@material-ui/core";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import useTransferSignedVAA from "../../hooks/useTransferSignedVAA";
import { selectIsRedeeming, selectTargetChain } from "../../store/selectors";
import { setIsRedeeming } from "../../store/transferSlice";
import { CHAIN_ID_ETH } from "../../utils/consts";
import redeemOn, { redeemOnEth } from "../../utils/redeemOn";

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
  const targetChain = useSelector(selectTargetChain);
  const { provider, signer } = useEthereumProvider();
  const signedVAA = useTransferSignedVAA();
  const isRedeeming = useSelector(selectIsRedeeming);
  const handleRedeemClick = useCallback(() => {
    if (
      targetChain === CHAIN_ID_ETH &&
      redeemOn[targetChain] === redeemOnEth &&
      signedVAA
    ) {
      dispatch(setIsRedeeming(true));
      redeemOnEth(provider, signer, signedVAA);
    }
  }, [dispatch, targetChain, provider, signer, signedVAA]);
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
