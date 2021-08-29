import { Button, makeStyles, MenuItem, TextField } from "@material-ui/core";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import { CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import useSyncTargetAddress from "../../hooks/useSyncTargetAddress";
import {
  selectTransferIsTargetComplete,
  selectTransferShouldLockFields,
  selectTransferSourceChain,
  selectTransferTargetAddressHex,
  selectTransferTargetAsset,
  selectTransferTargetBalanceString,
  selectTransferTargetChain,
} from "../../store/selectors";
import { incrementStep, setTargetChain } from "../../store/transferSlice";
import { hexToNativeString } from "../../utils/array";
import { CHAINS } from "../../utils/consts";
import KeyAndBalance from "../KeyAndBalance";
import SolanaCreateAssociatedAddress from "../SolanaCreateAssociatedAddress";

const useStyles = makeStyles((theme) => ({
  transferField: {
    marginTop: theme.spacing(5),
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
  const targetAddressHex = useSelector(selectTransferTargetAddressHex); // TODO: make readable
  const targetAsset = useSelector(selectTransferTargetAsset);
  const readableTargetAddress =
    hexToNativeString(targetAddressHex, targetChain) || "";
  const uiAmountString = useSelector(selectTransferTargetBalanceString);
  const isTargetComplete = useSelector(selectTransferIsTargetComplete);
  const shouldLockFields = useSelector(selectTransferShouldLockFields);
  useSyncTargetAddress(!shouldLockFields);
  const handleTargetChange = useCallback(
    (event) => {
      dispatch(setTargetChain(event.target.value));
    },
    [dispatch]
  );
  const handleNextClick = useCallback(
    (event) => {
      dispatch(incrementStep());
    },
    [dispatch]
  );
  return (
    <>
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
        label="Address"
        fullWidth
        className={classes.transferField}
        value={readableTargetAddress}
        disabled={true}
      />
      {targetChain === CHAIN_ID_SOLANA && targetAsset ? (
        <SolanaCreateAssociatedAddress
          mintAddress={targetAsset}
          readableTargetAddress={readableTargetAddress}
        />
      ) : null}
      <TextField
        label="Asset"
        fullWidth
        className={classes.transferField}
        value={targetAsset || ""}
        disabled={true}
      />
      <Button
        disabled={!isTargetComplete}
        onClick={handleNextClick}
        variant="contained"
        color="primary"
      >
        Next
      </Button>
    </>
  );
}

export default Target;
