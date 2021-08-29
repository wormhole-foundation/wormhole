import { useHandleCreateWrapped } from "../../hooks/useHandleCreateWrapped";
import ButtonWithLoader from "../ButtonWithLoader";

function Create() {
  const { handleClick, disabled, showLoader } = useHandleCreateWrapped();
  return (
    <ButtonWithLoader
      disabled={disabled}
      onClick={handleClick}
      showLoader={showLoader}
    >
      Create
    </ButtonWithLoader>
  );
}

export default Create;
