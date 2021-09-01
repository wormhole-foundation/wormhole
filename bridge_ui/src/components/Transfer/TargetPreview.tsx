import { makeStyles, Typography } from "@material-ui/core";
import { useSelector } from "react-redux";
import {
  selectTransferTargetAddressHex,
  selectTransferTargetChain,
} from "../../store/selectors";
import { hexToNativeString } from "../../utils/array";
import { CHAINS_BY_ID } from "../../utils/consts";
import { shortenAddress } from "../../utils/solana";

const useStyles = makeStyles((theme) => ({
  description: {
    textAlign: "center",
  },
}));

export default function TargetPreview() {
  const classes = useStyles();
  const targetChain = useSelector(selectTransferTargetChain);
  const targetAddress = useSelector(selectTransferTargetAddressHex); //TODO convert to readable
  const targetAddressNative = hexToNativeString(targetAddress, targetChain);

  const explainerString = targetAddressNative
    ? `to ${shortenAddress(targetAddressNative)} on ${
        CHAINS_BY_ID[targetChain].name
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
