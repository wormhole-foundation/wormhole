import { makeStyles, Typography } from "@material-ui/core";
import numeral from "numeral";
import { useSelector } from "react-redux";
import {
  selectSourceWalletAddress,
  selectTransferAmount,
  selectTransferRelayerFee,
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccount,
} from "../../store/selectors";
import { CHAINS_BY_ID } from "../../utils/consts";
import SmartAddress from "../SmartAddress";

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
  const sourceWalletAddress = useSelector(selectSourceWalletAddress);
  const sourceAmount = useSelector(selectTransferAmount);
  const relayerFee = useSelector(selectTransferRelayerFee);

  const explainerContent =
    sourceChain && sourceParsedTokenAccount ? (
      <>
        <span>
          You will transfer {sourceAmount}{" "}
          {relayerFee
            ? `(+~${numeral(relayerFee).format("0.00")} relayer fee)`
            : ""}
        </span>
        <SmartAddress
          chainId={sourceChain}
          parsedTokenAccount={sourceParsedTokenAccount}
        />
        {sourceWalletAddress ? (
          <>
            <span>from</span>
            <SmartAddress chainId={sourceChain} address={sourceWalletAddress} />
          </>
        ) : null}
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
    </>
  );
}
