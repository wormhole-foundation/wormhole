import { makeStyles, Typography } from "@material-ui/core";
import { useSelector } from "react-redux";
import {
  selectNFTSourceChain,
  selectNFTSourceParsedTokenAccount,
} from "../../store/selectors";
import { CHAINS_BY_ID } from "../../utils/consts";
import { shortenAddress } from "../../utils/solana";
import NFTViewer from "../TokenSelectors/NFTViewer";

const useStyles = makeStyles((theme) => ({
  description: {
    textAlign: "center",
  },
}));

export default function SourcePreview() {
  const classes = useStyles();
  const sourceChain = useSelector(selectNFTSourceChain);
  const sourceParsedTokenAccount = useSelector(
    selectNFTSourceParsedTokenAccount
  );

  const explainerString = sourceParsedTokenAccount
    ? `You will transfer 1 NFT of ${shortenAddress(
        sourceParsedTokenAccount?.mintKey
      )}, from ${shortenAddress(sourceParsedTokenAccount?.publicKey)} on ${
        CHAINS_BY_ID[sourceChain].name
      }`
    : "Step complete.";

  return (
    <>
      <Typography
        component="div"
        variant="subtitle2"
        className={classes.description}
      >
        {explainerString}
      </Typography>
      {sourceParsedTokenAccount ? (
        <NFTViewer value={sourceParsedTokenAccount} />
      ) : null}
    </>
  );
}
