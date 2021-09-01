import { Alert } from "@material-ui/lab";
import { useSelector } from "react-redux";
import { useHandleTransfer } from "../../hooks/useHandleTransfer";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import {
  selectTransferSourceChain,
  selectTransferTargetError,
} from "../../store/selectors";
import { CHAINS_BY_ID } from "../../utils/consts";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import StepDescription from "../StepDescription";

function Send() {
  const { handleClick, disabled, showLoader } = useHandleTransfer();
  const sourceChain = useSelector(selectTransferSourceChain);
  const error = useSelector(selectTransferTargetError);
  const { isReady, statusMessage } = useIsWalletReady(sourceChain);
  return (
    <>
      <StepDescription>Transfer the tokens to the worm bridge.</StepDescription>
      <KeyAndBalance chainId={sourceChain} />
      <Alert severity="warning">
        This will initiate the transfer on {CHAINS_BY_ID[sourceChain].name} and
        wait for finalization. If you navigate away from this page before
        completing Step 4, you will have to perform the recovery workflow to
        complete the transfer.
      </Alert>
      <ButtonWithLoader
        disabled={!isReady || disabled}
        onClick={handleClick}
        showLoader={showLoader}
        error={statusMessage || error}
      >
        Transfer
      </ButtonWithLoader>
    </>
  );
}

export default Send;
