import { useSelector } from "react-redux";
import { useHandleRedeem } from "../../hooks/useHandleRedeem";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import { selectTransferTargetChain } from "../../store/selectors";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";

function Redeem() {
  const { handleClick, disabled, showLoader } = useHandleRedeem();
  const targetChain = useSelector(selectTransferTargetChain);
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
        Redeem
      </ButtonWithLoader>
    </>
  );
}

export default Redeem;
