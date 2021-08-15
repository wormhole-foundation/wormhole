import { CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { Button, makeStyles, MenuItem, TextField } from "@material-ui/core";
import { PublicKey } from "@solana/web3.js";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import {
  selectTransferIsSourceAssetWormholeWrapped,
  selectTransferIsTargetComplete,
  selectTransferShouldLockFields,
  selectTransferSourceChain,
  selectTransferTargetAsset,
  selectTransferTargetBalanceString,
  selectTransferTargetChain,
} from "../../store/selectors";
import { incrementStep, setTargetChain } from "../../store/transferSlice";
import { hexToUint8Array } from "../../utils/array";
import { CHAINS } from "../../utils/consts";
import KeyAndBalance from "../KeyAndBalance";

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
  const targetAsset = useSelector(selectTransferTargetAsset);
  const isSourceAssetWormholeWrapped = useSelector(
    selectTransferIsSourceAssetWormholeWrapped
  );
  // TODO: wrapped stuff in hex, but native in not hex?
  const readableTargetAsset =
    isSourceAssetWormholeWrapped &&
    targetChain === CHAIN_ID_SOLANA &&
    targetAsset
      ? new PublicKey(hexToUint8Array(targetAsset)).toString()
      : targetAsset || "";
  // TODO: why doesn't this show up for solana wrapped?
  const uiAmountString = useSelector(selectTransferTargetBalanceString);
  const isTargetComplete = useSelector(selectTransferIsTargetComplete);
  const shouldLockFields = useSelector(selectTransferShouldLockFields);
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
        placeholder="Asset"
        fullWidth
        className={classes.transferField}
        value={readableTargetAsset}
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
