import { useSelector } from "react-redux";
import { useHandleAttest } from "../../hooks/useHandleAttest";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import { selectAttestSourceChain } from "../../store/selectors";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";

function Send() {
  const { handleClick, disabled, showLoader } = useHandleAttest();
  const sourceChain = useSelector(selectAttestSourceChain);
  const { isReady, statusMessage } = useIsWalletReady(sourceChain);

  return (
    <>
      <KeyAndBalance chainId={sourceChain} />
      <ButtonWithLoader
        disabled={!isReady || disabled}
        onClick={handleClick}
        showLoader={showLoader}
        error={statusMessage}
      >
        Attest
      </ButtonWithLoader>
    </>
  );
}

export default Send;
