import { useSelector } from "react-redux";
import { useHandleCreateWrapped } from "../../hooks/useHandleCreateWrapped";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import { selectAttestTargetChain } from "../../store/selectors";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import WaitingForWalletMessage from "./WaitingForWalletMessage";

function Create() {
  const { handleClick, disabled, showLoader } = useHandleCreateWrapped();
  const targetChain = useSelector(selectAttestTargetChain);
  const { isReady, statusMessage } = useIsWalletReady(targetChain);
  return (
    <>
      <KeyAndBalance chainId={targetChain} />
      <ButtonWithLoader
        disabled={!isReady || disabled}
        onClick={handleClick}
        showLoader={showLoader}
        error={statusMessage}
      >
        Create
      </ButtonWithLoader>
      <WaitingForWalletMessage />
    </>
  );
}

export default Create;
