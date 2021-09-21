import { makeStyles, Typography } from "@material-ui/core";
import { useSelector } from "react-redux";
import {
  selectNFTSourceChain,
  selectNFTSourceParsedTokenAccount,
} from "../../store/selectors";
import { CHAINS_BY_ID } from "../../utils/consts";
import SmartAddress from "../SmartAddress";
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

  const explainerContent =
    sourceChain && sourceParsedTokenAccount ? (
      <>
        <span>You will transfer 1 NFT of</span>
        <SmartAddress
          chainId={sourceChain}
          parsedTokenAccount={sourceParsedTokenAccount}
        />
        <span>from</span>
        <SmartAddress
          chainId={sourceChain}
          address={sourceParsedTokenAccount?.publicKey}
        />
        <span>on {CHAINS_BY_ID[sourceChain].name}</span>
      </>
    ) : (
      ""
    );

  return (
    <>
      <Typography
        component="div"
        variant="subtitle2"
        className={classes.description}
      >
        {explainerContent}
      </Typography>
      {sourceParsedTokenAccount ? (
        <NFTViewer value={sourceParsedTokenAccount} chainId={sourceChain} />
      ) : null}
    </>
  );
}
