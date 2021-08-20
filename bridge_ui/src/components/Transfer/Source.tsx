import { Button, makeStyles, MenuItem, TextField } from "@material-ui/core";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import {
  selectTransferAmount,
  selectTransferIsSourceComplete,
  selectTransferShouldLockFields,
  selectTransferSourceAsset,
  selectTransferSourceBalanceString,
  selectTransferSourceChain,
} from "../../store/selectors";
import {
  incrementStep,
  setAmount,
  setSourceAsset,
  setSourceChain,
} from "../../store/transferSlice";
import { CHAINS } from "../../utils/consts";
import KeyAndBalance from "../KeyAndBalance";

const useStyles = makeStyles((theme) => ({
  transferField: {
    marginTop: theme.spacing(5),
  },
}));

function Source() {
  const classes = useStyles();
  const dispatch = useDispatch();
  const sourceChain = useSelector(selectTransferSourceChain);
  const sourceAsset = useSelector(selectTransferSourceAsset);
  const uiAmountString = useSelector(selectTransferSourceBalanceString);
  const amount = useSelector(selectTransferAmount);
  const isSourceComplete = true; // useSelector(selectTransferIsSourceComplete);
  const shouldLockFields = useSelector(selectTransferShouldLockFields);
  const handleSourceChange = useCallback(
    (event) => {
      dispatch(setSourceChain(event.target.value));
    },
    [dispatch]
  );
  const handleAssetChange = useCallback(
    (event) => {
      dispatch(setSourceAsset(event.target.value));
    },
    [dispatch]
  );
  const handleAmountChange = useCallback(
    (event) => {
      dispatch(setAmount(event.target.value));
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
        value={sourceChain}
        onChange={handleSourceChange}
        disabled={shouldLockFields}
      >
        {CHAINS.map(({ id, name }) => (
          <MenuItem key={id} value={id}>
            {name}
          </MenuItem>
        ))}
      </TextField>
      <KeyAndBalance chainId={sourceChain} balance={uiAmountString} />
      {/* TODO: token list for eth, check own */}
      <TextField
        placeholder="Asset"
        fullWidth
        className={classes.transferField}
        value={sourceAsset}
        onChange={handleAssetChange}
        disabled={shouldLockFields}
      />
      <TextField
        placeholder="Amount"
        type="number"
        fullWidth
        className={classes.transferField}
        value={amount}
        onChange={handleAmountChange}
        disabled={shouldLockFields}
      />
      <Button
        disabled={!isSourceComplete}
        onClick={handleNextClick}
        variant="contained"
        color="primary"
      >
        Next
      </Button>
    </>
  );
}

export default Source;
