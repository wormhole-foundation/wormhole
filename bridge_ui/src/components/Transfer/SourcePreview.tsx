import { makeStyles, Typography } from "@material-ui/core";
import { useSelector } from "react-redux";
import {
  selectTransferAmount,
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccount,
} from "../../store/selectors";
import { CHAINS_BY_ID } from "../../utils/consts";
import { shortenAddress } from "../../utils/solana";
import TokenBlacklistWarning from "./TokenBlacklistWarning";

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

  const tokenExplainer = !sourceParsedTokenAccount
    ? ""
    : sourceParsedTokenAccount.isNativeAsset
    ? sourceParsedTokenAccount.symbol
    : `token${plural ? "s" : ""} of ${
        sourceParsedTokenAccount.symbol ||
        shortenAddress(sourceParsedTokenAccount.mintKey)
      }`;

  const explainerString = sourceParsedTokenAccount
    ? `You will transfer ${sourceAmount} ${tokenExplainer}, from ${shortenAddress(
        sourceParsedTokenAccount?.publicKey
      )} on ${CHAINS_BY_ID[sourceChain].name}`
    : "";

  return (
    <>
      <Typography
        component="div"
        variant="subtitle2"
        className={classes.description}
      >
        {explainerString}
      </Typography>
      <TokenBlacklistWarning
        sourceChain={sourceChain}
        tokenAddress={sourceParsedTokenAccount?.mintKey}
        symbol={sourceParsedTokenAccount?.symbol}
      />
    </>
  );
}
