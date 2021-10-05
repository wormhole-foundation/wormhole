import { ChainId } from "@certusone/wormhole-sdk";
import { makeStyles, Typography } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import useIsWalletReady from "../hooks/useIsWalletReady";
import useTransactionFees from "../hooks/useTransactionFees";
import { getDefaultNativeCurrencySymbol } from "../utils/consts";

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
  const warningMessage = `This wallet has a very low ${getDefaultNativeCurrencySymbol(
    chainId
  )} balance and may not be able to pay for the upcoming transaction fees.`;

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
