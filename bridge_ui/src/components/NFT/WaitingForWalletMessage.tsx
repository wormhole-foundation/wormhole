import { CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
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

const useStyles = makeStyles((theme) => ({
  message: {
    color: theme.palette.warning.light,
    marginTop: theme.spacing(1),
    textAlign: "center",
  },
}));

const WAITING_FOR_WALLET = "Waiting for wallet approval (likely in a popup)...";

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
      {WAITING_FOR_WALLET}{" "}
      {targetChain === CHAIN_ID_SOLANA && isRedeeming
        ? "Note: there will be several transactions"
        : sourceChain === CHAIN_ID_ETH && isSending
        ? "Note: there will be two transactions"
        : null}
    </Typography>
  ) : null;
}
