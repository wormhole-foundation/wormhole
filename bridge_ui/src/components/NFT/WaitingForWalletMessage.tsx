import { CHAIN_ID_SOLANA, isEVMChain } from "@certusone/wormhole-sdk";
import { makeStyles, Typography } from "@material-ui/core";
import { useSelector } from "react-redux";
import {
  selectNFTIsRedeeming,
  selectNFTIsSending,
  selectNFTRedeemTx,
  selectNFTSourceChain,
  selectNFTTargetChain,
  selectNFTTransferTx,
} from "../../store/selectors";
import { WAITING_FOR_WALLET_AND_CONF } from "../Transfer/WaitingForWalletMessage";

const useStyles = makeStyles((theme) => ({
  message: {
    color: theme.palette.warning.light,
    marginTop: theme.spacing(1),
    textAlign: "center",
  },
}));

export default function WaitingForWalletMessage() {
  const classes = useStyles();
  const sourceChain = useSelector(selectNFTSourceChain);
  const isSending = useSelector(selectNFTIsSending);
  const transferTx = useSelector(selectNFTTransferTx);
  const targetChain = useSelector(selectNFTTargetChain);
  const isRedeeming = useSelector(selectNFTIsRedeeming);
  const redeemTx = useSelector(selectNFTRedeemTx);
  const showWarning = (isSending && !transferTx) || (isRedeeming && !redeemTx);
  return showWarning ? (
    <Typography className={classes.message} variant="body2">
      {WAITING_FOR_WALLET_AND_CONF}{" "}
      {targetChain === CHAIN_ID_SOLANA && isRedeeming
        ? "Note: there will be several transactions"
        : isEVMChain(sourceChain) && isSending
        ? "Note: there will be two transactions"
        : null}
    </Typography>
  ) : null;
}
