import { useSelector } from "react-redux";
import { useHandleTransfer } from "../../hooks/useHandleTransfer";
import { selectTransferSourceChain } from "../../store/selectors";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";

function Send() {
  const { handleClick, disabled, showLoader } = useHandleTransfer();
  const sourceChain = useSelector(selectTransferSourceChain);
  return (
    <>
      <KeyAndBalance chainId={sourceChain} />
      <ButtonWithLoader
        disabled={disabled}
        onClick={handleClick}
        showLoader={showLoader}
      >
        Transfer
      </ButtonWithLoader>
    </>
  );
}

export default Send;
