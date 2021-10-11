import { Checkbox, FormControlLabel } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { ethers } from "ethers";
import { parseUnits } from "ethers/lib/utils";
import { useCallback, useMemo, useState } from "react";
import { useSelector } from "react-redux";
import useAllowance from "../../hooks/useAllowance";
import { useHandleTransfer } from "../../hooks/useHandleTransfer";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import {
  selectSourceWalletAddress,
  selectTransferAmount,
  selectTransferIsSendComplete,
  selectTransferSourceAsset,
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccount,
  selectTransferTargetError,
  selectTransferTransferTx,
} from "../../store/selectors";
import { CHAINS_BY_ID } from "../../utils/consts";
import { isEVMChain } from "../../utils/ethereum";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import ShowTx from "../ShowTx";
import StepDescription from "../StepDescription";
import TransactionProgress from "../TransactionProgress";
import WaitingForWalletMessage from "./WaitingForWalletMessage";

function Send() {
  const { handleClick, disabled, showLoader } = useHandleTransfer();

  const sourceChain = useSelector(selectTransferSourceChain);
  const sourceAsset = useSelector(selectTransferSourceAsset);
  const sourceAmount = useSelector(selectTransferAmount);
  const sourceDecimals = useSelector(
    selectTransferSourceParsedTokenAccount
  )?.decimals;
  const sourceAmountParsed =
    sourceDecimals !== undefined &&
    sourceDecimals !== null &&
    sourceAmount &&
    parseUnits(sourceAmount, sourceDecimals).toBigInt();
  const oneParsed =
    sourceDecimals !== undefined &&
    sourceDecimals !== null &&
    parseUnits("1", sourceDecimals).toBigInt();
  const transferTx = useSelector(selectTransferTransferTx);
  const isSendComplete = useSelector(selectTransferIsSendComplete);

  const error = useSelector(selectTransferTargetError);
  const [allowanceError, setAllowanceError] = useState("");
  const { isReady, statusMessage, walletAddress } =
    useIsWalletReady(sourceChain);
  const sourceWalletAddress = useSelector(selectSourceWalletAddress);
  //The chain ID compare is handled implicitly, as the isWalletReady hook should report !isReady if the wallet is on the wrong chain.
  const isWrongWallet =
    sourceWalletAddress &&
    walletAddress &&
    sourceWalletAddress !== walletAddress;
  const [shouldApproveUnlimited, setShouldApproveUnlimited] = useState(false);
  const toggleShouldApproveUnlimited = useCallback(
    () => setShouldApproveUnlimited(!shouldApproveUnlimited),
    [shouldApproveUnlimited]
  );

  const {
    sufficientAllowance,
    isAllowanceFetching,
    isApproveProcessing,
    approveAmount,
  } = useAllowance(sourceChain, sourceAsset, sourceAmountParsed || undefined);

  const approveButtonNeeded = isEVMChain(sourceChain) && !sufficientAllowance;
  const notOne = shouldApproveUnlimited || sourceAmountParsed !== oneParsed;
  const isDisabled =
    !isReady ||
    isWrongWallet ||
    disabled ||
    isAllowanceFetching ||
    isApproveProcessing;
  const errorMessage = isWrongWallet
    ? "A different wallet is connected than in Step 1."
    : statusMessage || error || allowanceError || undefined;

  const approveExactAmount = useMemo(() => {
    return () => {
      setAllowanceError("");
      approveAmount(BigInt(sourceAmountParsed)).then(
        () => {
          setAllowanceError("");
        },
        (error) => setAllowanceError("Failed to approve the token transfer.")
      );
    };
  }, [approveAmount, sourceAmountParsed]);
  const approveUnlimited = useMemo(() => {
    return () => {
      setAllowanceError("");
      approveAmount(ethers.constants.MaxUint256.toBigInt()).then(
        () => {
          setAllowanceError("");
        },
        (error) => setAllowanceError("Failed to approve the token transfer.")
      );
    };
  }, [approveAmount]);

  return (
    <>
      <StepDescription>
        Transfer the tokens to the Wormhole Token Bridge.
      </StepDescription>
      <KeyAndBalance chainId={sourceChain} />
      <Alert severity="info" variant="outlined">
        This will initiate the transfer on {CHAINS_BY_ID[sourceChain].name} and
        wait for finalization. If you navigate away from this page before
        completing Step 4, you will have to perform the recovery workflow to
        complete the transfer.
      </Alert>
      {approveButtonNeeded ? (
        <>
          <FormControlLabel
            control={
              <Checkbox
                checked={shouldApproveUnlimited}
                onChange={toggleShouldApproveUnlimited}
                color="primary"
              />
            }
            label="Approve Unlimited Tokens"
          />
          <ButtonWithLoader
            disabled={isDisabled}
            onClick={
              shouldApproveUnlimited ? approveUnlimited : approveExactAmount
            }
            showLoader={isAllowanceFetching || isApproveProcessing}
            error={errorMessage}
          >
            {"Approve " +
              (shouldApproveUnlimited ? "Unlimited" : sourceAmount) +
              ` Token${notOne ? "s" : ""}`}
          </ButtonWithLoader>
        </>
      ) : (
        <ButtonWithLoader
          disabled={isDisabled}
          onClick={handleClick}
          showLoader={showLoader}
          error={errorMessage}
        >
          Transfer
        </ButtonWithLoader>
      )}
      <WaitingForWalletMessage />
      {transferTx ? <ShowTx chainId={sourceChain} tx={transferTx} /> : null}
      <TransactionProgress
        chainId={sourceChain}
        tx={transferTx}
        isSendComplete={isSendComplete}
      />
    </>
  );
}

export default Send;
