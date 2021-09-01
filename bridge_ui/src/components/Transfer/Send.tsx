import { useSelector } from "react-redux";
import { useHandleTransfer } from "../../hooks/useHandleTransfer";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import {
  selectTransferSourceChain,
  selectTransferTargetError,
} from "../../store/selectors";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";

function Send() {
  const { handleClick, disabled, showLoader } = useHandleTransfer();
  const sourceChain = useSelector(selectTransferSourceChain);
  const error = useSelector(selectTransferTargetError);
  const { isReady, statusMessage } = useIsWalletReady(sourceChain);
  return (
    <>
      <KeyAndBalance chainId={sourceChain} />
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
