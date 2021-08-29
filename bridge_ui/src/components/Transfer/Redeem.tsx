import { useSelector } from "react-redux";
import { useHandleRedeem } from "../../hooks/useHandleRedeem";
import { selectTransferTargetChain } from "../../store/selectors";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";

function Redeem() {
  const { handleClick, disabled, showLoader } = useHandleRedeem();
  const targetChain = useSelector(selectTransferTargetChain);
  return (
    <>
      <KeyAndBalance chainId={targetChain} />
      <ButtonWithLoader
        disabled={disabled}
        onClick={handleClick}
        showLoader={showLoader}
      >
        Redeem
      </ButtonWithLoader>
    </>
  );
}

export default Redeem;
