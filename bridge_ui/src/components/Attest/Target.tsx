import { Button, MenuItem, TextField } from "@material-ui/core";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import {
  selectAttestIsTargetComplete,
  selectAttestShouldLockFields,
  selectAttestSourceChain,
  selectAttestTargetChain,
} from "../../store/selectors";
import { incrementStep, setTargetChain } from "../../store/attestSlice";
import { CHAINS } from "../../utils/consts";
import KeyAndBalance from "../KeyAndBalance";

function Target() {
  const dispatch = useDispatch();
  const sourceChain = useSelector(selectAttestSourceChain);
  const chains = useMemo(
    () => CHAINS.filter((c) => c.id !== sourceChain),
    [sourceChain]
  );
  const targetChain = useSelector(selectAttestTargetChain);
  const isTargetComplete = useSelector(selectAttestIsTargetComplete);
  const shouldLockFields = useSelector(selectAttestShouldLockFields);
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
      {/* TODO: determine "to" token address */}
      <KeyAndBalance chainId={targetChain} />
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
