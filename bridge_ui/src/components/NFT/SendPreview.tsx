import { makeStyles, Typography } from "@material-ui/core";
import { useSelector } from "react-redux";
import {
  selectNFTSourceChain,
  selectNFTTransferTx,
} from "../../store/selectors";
import ShowTx from "../ShowTx";

const useStyles = makeStyles((theme) => ({
  description: {
    textAlign: "center",
  },
  tx: {
    marginTop: theme.spacing(1),
    textAlign: "center",
  },
  viewButton: {
    marginTop: theme.spacing(1),
  },
}));

export default function SendPreview() {
  const classes = useStyles();
  const sourceChain = useSelector(selectNFTSourceChain);
  const transferTx = useSelector(selectNFTTransferTx);

  const explainerString = "The NFT has been sent!";

  return (
    <>
      <Typography
        component="div"
        variant="subtitle2"
        className={classes.description}
      >
        {explainerString}
      </Typography>
      {transferTx ? <ShowTx chainId={sourceChain} tx={transferTx} /> : null}
    </>
  );
}
