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
  selectTransferOriginAsset,
  selectTransferOriginChain,
  selectTransferSourceAsset,
  selectTransferTargetChain,
} from "../../store/selectors";
import {
  ChainId,
  CHAIN_ID_TERRA2,
  hexToNativeAssetString,
} from "@certusone/wormhole-sdk";

export function RegisterNowButtonCore({
  originChain,
  originAsset,
  targetChain,
}: {
  originChain: ChainId | undefined;
  originAsset: string | undefined;
  targetChain: ChainId;
}) {
  const dispatch = useDispatch();
  const history = useHistory();
  // user might be in the middle of a different attest
  const signedVAAHex = useSelector(selectAttestSignedVAAHex);
  const sourceAsset = useSelector(selectTransferSourceAsset);
  const canSwitch = originChain && originAsset && !signedVAAHex;
  const handleClick = useCallback(() => {
    const nativeAsset = originChain
      ? originChain === CHAIN_ID_TERRA2
        ? sourceAsset // use the preimage address for terra2
        : hexToNativeAssetString(originAsset, originChain)
      : undefined;
    if (originChain && originAsset && nativeAsset && canSwitch) {
      dispatch(setSourceChain(originChain));
      dispatch(setSourceAsset(nativeAsset));
      dispatch(setTargetChain(targetChain));
      dispatch(setStep(2));
      history.push("/register");
    }
  }, [
    dispatch,
    canSwitch,
    originChain,
    originAsset,
    targetChain,
    history,
    sourceAsset,
  ]);
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

export default function RegisterNowButton() {
  const originChain = useSelector(selectTransferOriginChain);
  const originAsset = useSelector(selectTransferOriginAsset);
  const targetChain = useSelector(selectTransferTargetChain);
  return (
    <RegisterNowButtonCore
      originChain={originChain}
      originAsset={originAsset}
      targetChain={targetChain}
    />
  );
}
