import { makeStyles, Typography } from "@material-ui/core";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import {
  selectTransferRedeemTx,
  selectTransferTargetChain,
} from "../../store/selectors";
import { reset } from "../../store/transferSlice";
import ButtonWithLoader from "../ButtonWithLoader";
import ShowTx from "../ShowTx";
import AddToMetamask from "./AddToMetamask";
import FeaturedMarkets from "./FeaturedMarkets";

const useStyles = makeStyles((theme) => ({
  description: {
    textAlign: "center",
  },
}));

export default function RedeemPreview() {
  const classes = useStyles();
  const dispatch = useDispatch();
  const targetChain = useSelector(selectTransferTargetChain);
  const redeemTx = useSelector(selectTransferRedeemTx);
  const handleResetClick = useCallback(() => {
    dispatch(reset());
  }, [dispatch]);

  const explainerString =
    "Success! The redeem transaction was submitted. The tokens will become available once the transaction confirms.";

  return (
    <>
      <Typography
        component="div"
        variant="subtitle2"
        className={classes.description}
      >
        {explainerString}
      </Typography>
      {redeemTx ? <ShowTx chainId={targetChain} tx={redeemTx} /> : null}
      <AddToMetamask />
      <FeaturedMarkets />
      <ButtonWithLoader onClick={handleResetClick}>
        Transfer More Tokens!
      </ButtonWithLoader>
    </>
  );
}
