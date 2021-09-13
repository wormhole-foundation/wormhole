import { Alert } from "@material-ui/lab";
import { useSelector } from "react-redux";
import { useHandleTransfer } from "../../hooks/useHandleTransfer";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import {
  selectSourceWalletAddress,
  selectTransferSourceChain,
  selectTransferTargetError,
} from "../../store/selectors";
import { CHAINS_BY_ID } from "../../utils/consts";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import StepDescription from "../StepDescription";
import TransferProgress from "../TransferProgress";
import WaitingForWalletMessage from "./WaitingForWalletMessage";

function Send() {
  const { handleClick, disabled, showLoader } = useHandleTransfer();
  const sourceChain = useSelector(selectTransferSourceChain);
  const error = useSelector(selectTransferTargetError);
  const { isReady, statusMessage, walletAddress } =
    useIsWalletReady(sourceChain);
  const sourceWalletAddress = useSelector(selectSourceWalletAddress);
  //The chain ID compare is handled implicitly, as the isWalletReady hook should report !isReady if the wallet is on the wrong chain.
  const isWrongWallet =
    sourceWalletAddress &&
    walletAddress &&
    sourceWalletAddress !== walletAddress;
  const isDisabled = !isReady || isWrongWallet || disabled;
  const errorMessage = isWrongWallet
    ? "A different wallet is connected than in Step 1."
    : statusMessage || error || undefined;
  return (
    <>
      <StepDescription>
        Transfer the tokens to the Wormhole Token Bridge.
      </StepDescription>
      <KeyAndBalance chainId={sourceChain} />
      <Alert severity="warning">
        This will initiate the transfer on {CHAINS_BY_ID[sourceChain].name} and
        wait for finalization. If you navigate away from this page before
        completing Step 4, you will have to perform the recovery workflow to
        complete the transfer.
      </Alert>
      <ButtonWithLoader
        disabled={isDisabled}
        onClick={handleClick}
        showLoader={showLoader}
        error={errorMessage}
      >
        Transfer
      </ButtonWithLoader>
      <WaitingForWalletMessage />
      <TransferProgress />
    </>
  );
}

export default Send;
