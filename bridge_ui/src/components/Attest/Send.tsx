import { useHandleAttest } from "../../hooks/useHandleAttest";
import ButtonWithLoader from "../ButtonWithLoader";

function Send() {
  const { handleClick, disabled, showLoader } = useHandleAttest();
  return (
    <ButtonWithLoader
      disabled={disabled}
      onClick={handleClick}
      showLoader={showLoader}
    >
      Attest
    </ButtonWithLoader>
  );
}

export default Send;
