import { CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { makeStyles, MenuItem, TextField } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import useSyncTargetAddress from "../../hooks/useSyncTargetAddress";
import {
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
import { hexToNativeString } from "../../utils/array";
import { CHAINS, CHAINS_BY_ID } from "../../utils/consts";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import LowBalanceWarning from "../LowBalanceWarning";
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
  const sourceChain = useSelector(selectTransferSourceChain);
  const chains = useMemo(
    () => CHAINS.filter((c) => c.id !== sourceChain),
    [sourceChain]
  );
  const targetChain = useSelector(selectTransferTargetChain);
  const targetAddressHex = useSelector(selectTransferTargetAddressHex);
  const targetAsset = useSelector(selectTransferTargetAsset);
  const readableTargetAddress =
    hexToNativeString(targetAddressHex, targetChain) || "";
  const uiAmountString = useSelector(selectTransferTargetBalanceString);
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
        select
        fullWidth
        value={targetChain}
        onChange={handleTargetChange}
        disabled={shouldLockFields}
      >
        {chains.map(({ id, name }) => (
          <MenuItem key={id} value={id}>
            {name}
          </MenuItem>
        ))}
      </TextField>
      <KeyAndBalance chainId={targetChain} balance={uiAmountString} />
      <TextField
        label="Recipient Address"
        fullWidth
        className={classes.transferField}
        value={readableTargetAddress}
        disabled={true}
      />
      {targetChain === CHAIN_ID_SOLANA && targetAsset ? (
        <SolanaCreateAssociatedAddress
          mintAddress={targetAsset}
          readableTargetAddress={readableTargetAddress}
          associatedAccountExists={associatedAccountExists}
          setAssociatedAccountExists={setAssociatedAccountExists}
        />
      ) : null}
      <TextField
        label="Token Address"
        fullWidth
        className={classes.transferField}
        value={targetAsset || ""}
        disabled={true}
      />
      <Alert severity="info" className={classes.alert}>
        You will have to pay transaction fees on{" "}
        {CHAINS_BY_ID[targetChain].name} to redeem your tokens.
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
