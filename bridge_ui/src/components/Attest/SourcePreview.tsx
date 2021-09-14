import { makeStyles, Typography } from "@material-ui/core";
import { useSelector } from "react-redux";
import {
  selectAttestSourceAsset,
  selectAttestSourceChain,
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
  const sourceChain = useSelector(selectAttestSourceChain);
  const sourceAsset = useSelector(selectAttestSourceAsset);

  const explainerString = sourceAsset
    ? `You will attest ${shortenAddress(sourceAsset)} on ${
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
