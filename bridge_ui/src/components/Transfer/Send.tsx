import { useSelector } from "react-redux";
import { useHandleTransfer } from "../../hooks/useHandleTransfer";
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
  return (
    <>
      <KeyAndBalance chainId={sourceChain} />
      <ButtonWithLoader
        disabled={disabled}
        onClick={handleClick}
        showLoader={showLoader}
        error={error}
      >
        Transfer
      </ButtonWithLoader>
    </>
  );
}

export default Send;
