import { CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { makeStyles, Typography } from "@material-ui/core";
import { useSelector } from "react-redux";
import {
  selectAttestAttestTx,
  selectAttestCreateTx,
  selectAttestIsCreating,
  selectAttestIsSending,
  selectAttestTargetChain,
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
  const isSending = useSelector(selectAttestIsSending);
  const attestTx = useSelector(selectAttestAttestTx);
  const targetChain = useSelector(selectAttestTargetChain);
  const isCreating = useSelector(selectAttestIsCreating);
  const createTx = useSelector(selectAttestCreateTx);
  const showWarning = (isSending && !attestTx) || (isCreating && !createTx);
  return showWarning ? (
    <Typography className={classes.message} variant="body2">
      {WAITING_FOR_WALLET_AND_CONF}{" "}
      {targetChain === CHAIN_ID_SOLANA && isCreating
        ? "Note: there will be several transactions"
        : null}
    </Typography>
  ) : null;
}
