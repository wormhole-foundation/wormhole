import { Button } from "@material-ui/core";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useHistory } from "react-router-dom";
import {
  setSourceAsset,
  setSourceChain,
  setStep,
  setTargetChain,
} from "../../store/attestSlice";
import {
  selectAttestSignedVAAHex,
  selectTransferSourceAsset,
  selectTransferSourceChain,
  selectTransferTargetChain,
} from "../../store/selectors";

export default function RegisterNowButton() {
  const dispatch = useDispatch();
  const history = useHistory();
  const sourceChain = useSelector(selectTransferSourceChain);
  const sourceAsset = useSelector(selectTransferSourceAsset);
  const targetChain = useSelector(selectTransferTargetChain);
  // user might be in the middle of a different attest
  const signedVAAHex = useSelector(selectAttestSignedVAAHex);
  const canSwitch = sourceAsset && !signedVAAHex;
  const handleClick = useCallback(() => {
    if (sourceAsset && canSwitch) {
      dispatch(setSourceChain(sourceChain));
      dispatch(setSourceAsset(sourceAsset));
      dispatch(setTargetChain(targetChain));
      dispatch(setStep(2));
      history.push("/register");
    }
  }, [dispatch, canSwitch, sourceChain, sourceAsset, targetChain, history]);
  if (!canSwitch) return null;
  return (
    <Button
      variant="outlined"
      size="small"
      style={{ display: "block", margin: "4px auto 0px" }}
      onClick={handleClick}
    >
      Register Now
    </Button>
  );
}
