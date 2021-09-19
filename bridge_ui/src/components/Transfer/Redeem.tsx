import { CHAIN_ID_ETH } from "@certusone/wormhole-sdk";
import { Checkbox, FormControlLabel } from "@material-ui/core";
import { useCallback, useState } from "react";
import { useSelector } from "react-redux";
import { useHandleRedeem } from "../../hooks/useHandleRedeem";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import {
  selectTransferTargetAsset,
  selectTransferTargetChain,
} from "../../store/selectors";
import { WETH_ADDRESS } from "../../utils/consts";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import StepDescription from "../StepDescription";
import WaitingForWalletMessage from "./WaitingForWalletMessage";

function Redeem() {
  const { handleClick, handleNativeClick, disabled, showLoader } =
    useHandleRedeem();
  const targetChain = useSelector(selectTransferTargetChain);
  const targetAssetHex = useSelector(selectTransferTargetAsset);
  const { isReady, statusMessage } = useIsWalletReady(targetChain);
  //TODO better check, probably involving a hook & the VAA
  const isNativeEligible =
    targetChain === CHAIN_ID_ETH &&
    targetAssetHex &&
    targetAssetHex.toLowerCase() === WETH_ADDRESS.toLowerCase();
  const [useNativeRedeem, setUseNativeRedeem] = useState(true);
  const toggleNativeRedeem = useCallback(() => {
    setUseNativeRedeem(!useNativeRedeem);
  }, [useNativeRedeem]);

  return (
    <>
      <StepDescription>Receive the tokens on the target chain</StepDescription>
      <KeyAndBalance chainId={targetChain} />
      {isNativeEligible && (
        <FormControlLabel
          control={
            <Checkbox
              checked={useNativeRedeem}
              onChange={toggleNativeRedeem}
              color="primary"
            />
          }
          label="Automatically unwrap to native currency"
        />
      )}

      <ButtonWithLoader
        disabled={!isReady || disabled}
        onClick={
          isNativeEligible && useNativeRedeem ? handleNativeClick : handleClick
        }
        showLoader={showLoader}
        error={statusMessage}
      >
        Redeem
      </ButtonWithLoader>
      <WaitingForWalletMessage />
    </>
  );
}

export default Redeem;
