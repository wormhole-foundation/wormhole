import { Button, MenuItem, TextField } from "@material-ui/core";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import { selectSourceChain, selectTargetChain } from "../../store/selectors";
import { incrementStep, setTargetChain } from "../../store/transferSlice";
import { CHAINS } from "../../utils/consts";
import KeyAndBalance from "../KeyAndBalance";

function Target() {
  const dispatch = useDispatch();
  const sourceChain = useSelector(selectSourceChain);
  const chains = useMemo(
    () => CHAINS.filter((c) => c.id !== sourceChain),
    [sourceChain]
  );
  const targetChain = useSelector(selectTargetChain);
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
      >
        {chains.map(({ id, name }) => (
          <MenuItem key={id} value={id}>
            {name}
          </MenuItem>
        ))}
      </TextField>
      {/* TODO: determine "to" token address */}
      <KeyAndBalance chainId={targetChain} />
      <Button onClick={handleNextClick} variant="contained" color="primary">
        Next
      </Button>
    </>
  );
}

export default Target;
