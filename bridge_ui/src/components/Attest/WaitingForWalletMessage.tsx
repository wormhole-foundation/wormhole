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
  const isSending = useSelector(selectAttestIsSending);
  const attestTx = useSelector(selectAttestAttestTx);
  const targetChain = useSelector(selectAttestTargetChain);
  const isCreating = useSelector(selectAttestIsCreating);
  const createTx = useSelector(selectAttestCreateTx);
  const showWarning = (isSending && !attestTx) || (isCreating && !createTx);
  return showWarning ? (
    <Typography className={classes.message} variant="body2">
      {WAITING_FOR_WALLET}{" "}
      {targetChain === CHAIN_ID_SOLANA && isCreating
        ? "Note: there will be several transactions"
        : null}
    </Typography>
  ) : null;
}
