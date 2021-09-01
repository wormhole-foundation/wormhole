import { makeStyles, Typography } from "@material-ui/core";
import { useSelector } from "react-redux";
import {
  selectTransferAmount,
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccount,
} from "../../store/selectors";
import { CHAINS_BY_ID } from "../../utils/consts";
import { shortenAddress } from "../../utils/solana";

const useStyles = makeStyles((theme) => ({
  description: {
    textAlign: "center",
  },
}));

export default function SourcePreview() {
  const classes = useStyles();
  const sourceChain = useSelector(selectTransferSourceChain);
  const sourceParsedTokenAccount = useSelector(
    selectTransferSourceParsedTokenAccount
  );
  const sourceAmount = useSelector(selectTransferAmount);

  const plural = parseInt(sourceAmount) !== 1;

  const explainerString = sourceParsedTokenAccount
    ? `You will transfer ${sourceAmount} token${
        plural ? "s" : ""
      } of ${shortenAddress(
        sourceParsedTokenAccount?.mintKey
      )}, from ${shortenAddress(sourceParsedTokenAccount?.publicKey)} on ${
        CHAINS_BY_ID[sourceChain].name
      }`
    : "Step complete.";

  return (
    <Typography
      component="div"
      variant="subtitle2"
      className={classes.description}
    >
      {explainerString}
    </Typography>
  );
}
