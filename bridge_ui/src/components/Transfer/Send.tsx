import {
  CHAIN_ID_SOLANA,
  isEVMChain,
  isTerraChain,
} from "@certusone/wormhole-sdk";
import { Checkbox, FormControlLabel } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { ethers } from "ethers";
import { formatUnits, parseUnits } from "ethers/lib/utils";
import { useCallback, useMemo, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import useAllowance from "../../hooks/useAllowance";
import { useHandleTransfer } from "../../hooks/useHandleTransfer";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import {
  selectSourceWalletAddress,
  selectTransferAmount,
  selectTransferIsSendComplete,
  selectTransferIsVAAPending,
  selectTransferRelayerFee,
  selectTransferSourceAsset,
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccount,
  selectTransferTargetError,
  selectTransferTransferTx,
} from "../../store/selectors";
import { reset } from "../../store/transferSlice";
import { CHAINS_BY_ID, CLUSTER } from "../../utils/consts";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import ShowTx from "../ShowTx";
import SolanaTPSWarning from "../SolanaTPSWarning";
import StepDescription from "../StepDescription";
import TerraFeeDenomPicker from "../TerraFeeDenomPicker";
import TransactionProgress from "../TransactionProgress";
import PendingVAAWarning from "./PendingVAAWarning";
import SendConfirmationDialog from "./SendConfirmationDialog";
import WaitingForWalletMessage from "./WaitingForWalletMessage";

function Send() {
  const dispatch = useDispatch();
  const { handleClick, disabled, showLoader } = useHandleTransfer();
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const handleTransferClick = useCallback(() => {
    setIsConfirmOpen(true);
  }, []);
  const handleConfirmClick = useCallback(() => {
    handleClick();
    setIsConfirmOpen(false);
  }, [handleClick]);
  const handleConfirmClose = useCallback(() => {
    setIsConfirmOpen(false);
  }, []);
  const handleResetClick = useCallback(() => {
    dispatch(reset());
  }, [dispatch]);

  const sourceChain = useSelector(selectTransferSourceChain);
  const sourceAsset = useSelector(selectTransferSourceAsset);
  const sourceAmount = useSelector(selectTransferAmount);
  const sourceParsedTokenAccount = useSelector(
    selectTransferSourceParsedTokenAccount
  );
  const relayerFee = useSelector(selectTransferRelayerFee);
  const sourceDecimals = sourceParsedTokenAccount?.decimals;
  const sourceIsNative = sourceParsedTokenAccount?.isNativeAsset;
  const baseAmountParsed =
    sourceDecimals !== undefined &&
    sourceDecimals !== null &&
    sourceAmount &&
    parseUnits(sourceAmount, sourceDecimals);
  const feeParsed =
    sourceDecimals !== undefined
      ? parseUnits(relayerFee || "0", sourceDecimals)
      : 0;
  const transferAmountParsed =
    baseAmountParsed && baseAmountParsed.add(feeParsed).toBigInt();
  const humanReadableTransferAmount =
    sourceDecimals !== undefined &&
    sourceDecimals !== null &&
    transferAmountParsed &&
    formatUnits(transferAmountParsed, sourceDecimals);
  const oneParsed =
    sourceDecimals !== undefined &&
    sourceDecimals !== null &&
    parseUnits("1", sourceDecimals).toBigInt();
  const transferTx = useSelector(selectTransferTransferTx);
  const isSendComplete = useSelector(selectTransferIsSendComplete);
  const isVAAPending = useSelector(selectTransferIsVAAPending);

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
  } = useAllowance(
    sourceChain,
    sourceAsset,
    transferAmountParsed || undefined,
    sourceIsNative
  );

  const approveButtonNeeded = isEVMChain(sourceChain) && !sufficientAllowance;
  const notOne = shouldApproveUnlimited || transferAmountParsed !== oneParsed;
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
      approveAmount(BigInt(transferAmountParsed)).then(
        () => {
          setAllowanceError("");
        },
        (error) => setAllowanceError("Failed to approve the token transfer.")
      );
    };
  }, [approveAmount, transferAmountParsed]);
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
      {isTerraChain(sourceChain) && (
        <TerraFeeDenomPicker disabled={disabled} chainId={sourceChain} />
      )}
      <Alert severity="info" variant="outlined">
        This will initiate the transfer on {CHAINS_BY_ID[sourceChain].name} and
        wait for finalization. If you navigate away from this page before
        completing Step 4, you will have to perform the recovery workflow to
        complete the transfer.
      </Alert>
      {sourceChain === CHAIN_ID_SOLANA && CLUSTER === "mainnet" && (
        <SolanaTPSWarning />
      )}
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
              (shouldApproveUnlimited
                ? "Unlimited"
                : humanReadableTransferAmount
                ? humanReadableTransferAmount
                : sourceAmount) +
              ` Token${notOne ? "s" : ""}`}
          </ButtonWithLoader>
        </>
      ) : (
        <>
          <ButtonWithLoader
            disabled={isDisabled}
            onClick={handleTransferClick}
            showLoader={showLoader && !isVAAPending}
            error={errorMessage}
          >
            Transfer
          </ButtonWithLoader>
          <SendConfirmationDialog
            open={isConfirmOpen}
            onClick={handleConfirmClick}
            onClose={handleConfirmClose}
          />
        </>
      )}
      <WaitingForWalletMessage />
      {transferTx ? <ShowTx chainId={sourceChain} tx={transferTx} /> : null}
      <TransactionProgress
        chainId={sourceChain}
        tx={transferTx}
        isSendComplete={isSendComplete || isVAAPending}
      />
      {isVAAPending ? (
        <>
          <PendingVAAWarning />
          <ButtonWithLoader onClick={handleResetClick}>
            Transfer More Tokens!
          </ButtonWithLoader>
        </>
      ) : null}
    </>
  );
}

export default Send;
