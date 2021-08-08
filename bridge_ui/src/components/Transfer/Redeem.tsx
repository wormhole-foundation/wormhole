import { Button, makeStyles } from "@material-ui/core";
import { useCallback } from "react";
import { useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import useTransferSignedVAA from "../../hooks/useTransferSignedVAA";
import { selectTargetChain } from "../../store/selectors";
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
  const classes = useStyles();
  const targetChain = useSelector(selectTargetChain);
  const { provider, signer } = useEthereumProvider();
  const signedVAA = useTransferSignedVAA();
  const handleRedeemClick = useCallback(() => {
    if (
      targetChain === CHAIN_ID_ETH &&
      redeemOn[targetChain] === redeemOnEth &&
      signedVAA
    ) {
      redeemOnEth(provider, signer, signedVAA);
    }
  }, [targetChain, provider, signer, signedVAA]);
  return (
    <Button
      color="primary"
      variant="contained"
      className={classes.transferButton}
      onClick={handleRedeemClick}
    >
      Redeem
    </Button>
  );
}

export default Redeem;
