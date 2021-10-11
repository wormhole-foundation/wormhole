import { CHAIN_ID_SOLANA, hexToNativeString } from "@certusone/wormhole-sdk";
import { makeStyles, MenuItem, TextField, Typography } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useBetaContext } from "../../contexts/BetaContext";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import useMetadata from "../../hooks/useMetadata";
import useSyncTargetAddress from "../../hooks/useSyncTargetAddress";
import { EthGasEstimateSummary } from "../../hooks/useTransactionFees";
import {
  selectTransferAmount,
  selectTransferIsTargetComplete,
  selectTransferShouldLockFields,
  selectTransferSourceChain,
  selectTransferTargetAddressHex,
  selectTransferTargetAsset,
  selectTransferTargetBalanceString,
  selectTransferTargetChain,
  selectTransferTargetError,
  UNREGISTERED_ERROR_MESSAGE,
} from "../../store/selectors";
import { incrementStep, setTargetChain } from "../../store/transferSlice";
import { BETA_CHAINS, CHAINS, CHAINS_BY_ID } from "../../utils/consts";
import { isEVMChain } from "../../utils/ethereum";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import LowBalanceWarning from "../LowBalanceWarning";
import SmartAddress from "../SmartAddress";
import SolanaCreateAssociatedAddress, {
  useAssociatedAccountExistsState,
} from "../SolanaCreateAssociatedAddress";
import StepDescription from "../StepDescription";
import RegisterNowButton from "./RegisterNowButton";

const useStyles = makeStyles((theme) => ({
  transferField: {
    marginTop: theme.spacing(5),
  },
  alert: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
}));

function Target() {
  const classes = useStyles();
  const dispatch = useDispatch();
  const isBeta = useBetaContext();
  const sourceChain = useSelector(selectTransferSourceChain);
  const chains = useMemo(
    () => CHAINS.filter((c) => c.id !== sourceChain),
    [sourceChain]
  );
  const targetChain = useSelector(selectTransferTargetChain);
  const targetAddressHex = useSelector(selectTransferTargetAddressHex);
  const targetAsset = useSelector(selectTransferTargetAsset);
  const targetAssetArrayed = useMemo(
    () => (targetAsset ? [targetAsset] : []),
    [targetAsset]
  );
  const metadata = useMetadata(targetChain, targetAssetArrayed);
  const tokenName =
    (targetAsset && metadata.data?.get(targetAsset)?.tokenName) || undefined;
  const symbol =
    (targetAsset && metadata.data?.get(targetAsset)?.symbol) || undefined;
  const logo =
    (targetAsset && metadata.data?.get(targetAsset)?.logo) || undefined;
  const readableTargetAddress =
    hexToNativeString(targetAddressHex, targetChain) || "";
  const uiAmountString = useSelector(selectTransferTargetBalanceString);
  const transferAmount = useSelector(selectTransferAmount);
  const error = useSelector(selectTransferTargetError);
  const isTargetComplete = useSelector(selectTransferIsTargetComplete);
  const shouldLockFields = useSelector(selectTransferShouldLockFields);
  const { statusMessage } = useIsWalletReady(targetChain);
  const { associatedAccountExists, setAssociatedAccountExists } =
    useAssociatedAccountExistsState(
      targetChain,
      targetAsset,
      readableTargetAddress
    );
  useSyncTargetAddress(!shouldLockFields);
  const handleTargetChange = useCallback(
    (event) => {
      dispatch(setTargetChain(event.target.value));
    },
    [dispatch]
  );
  const handleNextClick = useCallback(() => {
    dispatch(incrementStep());
  }, [dispatch]);
  return (
    <>
      <StepDescription>Select a recipient chain and address.</StepDescription>
      <TextField
        variant="outlined"
        select
        fullWidth
        value={targetChain}
        onChange={handleTargetChange}
        disabled={shouldLockFields}
      >
        {chains
          .filter(({ id }) => (isBeta ? true : !BETA_CHAINS.includes(id)))
          .map(({ id, name }) => (
            <MenuItem key={id} value={id}>
              {name}
            </MenuItem>
          ))}
      </TextField>
      <KeyAndBalance chainId={targetChain} balance={uiAmountString} />
      {readableTargetAddress ? (
        <>
          {targetAsset ? (
            <div className={classes.transferField}>
              <Typography variant="subtitle2">Bridged tokens:</Typography>
              <Typography component="div">
                <SmartAddress
                  chainId={targetChain}
                  address={targetAsset}
                  symbol={symbol}
                  tokenName={tokenName}
                  logo={logo}
                  variant="h6"
                />
                {`(Amount: ${transferAmount})`}
              </Typography>
            </div>
          ) : null}
          <div className={classes.transferField}>
            <Typography variant="subtitle2">Sent to:</Typography>
            <Typography component="div">
              <SmartAddress
                chainId={targetChain}
                address={readableTargetAddress}
                variant="h6"
              />
              {`(Current balance: ${uiAmountString || "0"})`}
            </Typography>
          </div>
        </>
      ) : null}
      {targetChain === CHAIN_ID_SOLANA && targetAsset ? (
        <SolanaCreateAssociatedAddress
          mintAddress={targetAsset}
          readableTargetAddress={readableTargetAddress}
          associatedAccountExists={associatedAccountExists}
          setAssociatedAccountExists={setAssociatedAccountExists}
        />
      ) : null}
      <Alert severity="info" variant="outlined" className={classes.alert}>
        <Typography>
          You will have to pay transaction fees on{" "}
          {CHAINS_BY_ID[targetChain].name} to redeem your tokens.
        </Typography>
        {isEVMChain(targetChain) && (
          <EthGasEstimateSummary methodType="transfer" chainId={targetChain} />
        )}
      </Alert>
      <LowBalanceWarning chainId={targetChain} />
      <ButtonWithLoader
        disabled={!isTargetComplete || !associatedAccountExists}
        onClick={handleNextClick}
        showLoader={false}
        error={statusMessage || error}
      >
        Next
      </ButtonWithLoader>
      {!statusMessage && error === UNREGISTERED_ERROR_MESSAGE ? (
        <RegisterNowButton />
      ) : null}
    </>
  );
}

export default Target;
