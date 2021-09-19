import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import { Typography } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { makeStyles } from "@material-ui/core";
import useTransactionFees from "../hooks/useTransactionFees";
import useIsWalletReady from "../hooks/useIsWalletReady";

const useStyles = makeStyles((theme) => ({
  alert: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
}));

function LowBalanceWarning({ chainId }: { chainId: ChainId }) {
  const classes = useStyles();
  const { isReady } = useIsWalletReady(chainId);
  const transactionFeeWarning = useTransactionFees(chainId);
  const displayWarning =
    isReady &&
    transactionFeeWarning.balanceString &&
    transactionFeeWarning.isSufficientBalance === false;
  const warningMessage = `This wallet has a very low ${
    chainId === CHAIN_ID_SOLANA ? "SOL" : chainId === CHAIN_ID_ETH ? "ETH" : ""
  } balance and may not be able to pay for the upcoming transaction fees.`;

  const content = (
    <Alert severity="warning" className={classes.alert}>
      <Typography variant="body1">{warningMessage}</Typography>
      <Typography variant="body1">
        {"Current balance: " + transactionFeeWarning.balanceString}
      </Typography>
    </Alert>
  );

  return displayWarning ? content : null;
}

export default LowBalanceWarning;
