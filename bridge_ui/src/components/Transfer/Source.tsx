import { Button, makeStyles, MenuItem, TextField } from "@material-ui/core";
import { Restore } from "@material-ui/icons";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import {
  selectTransferAmount,
  selectTransferIsSourceComplete,
  selectTransferShouldLockFields,
  selectTransferSourceBalanceString,
  selectTransferSourceChain,
  selectTransferSourceError,
} from "../../store/selectors";
import {
  incrementStep,
  setAmount,
  setSourceChain,
} from "../../store/transferSlice";
import { CHAINS } from "../../utils/consts";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import StepDescription from "../StepDescription";
import { TokenSelector } from "../TokenSelectors/SourceTokenSelector";

const useStyles = makeStyles((theme) => ({
  transferField: {
    marginTop: theme.spacing(5),
  },
}));

function Source({
  setIsRecoveryOpen,
}: {
  setIsRecoveryOpen: (open: boolean) => void;
}) {
  const classes = useStyles();
  const dispatch = useDispatch();
  const sourceChain = useSelector(selectTransferSourceChain);
  const uiAmountString = useSelector(selectTransferSourceBalanceString);
  const amount = useSelector(selectTransferAmount);
  const error = useSelector(selectTransferSourceError);
  const isSourceComplete = useSelector(selectTransferIsSourceComplete);
  const shouldLockFields = useSelector(selectTransferShouldLockFields);
  const { isReady, statusMessage } = useIsWalletReady(sourceChain);
  const handleSourceChange = useCallback(
    (event) => {
      dispatch(setSourceChain(event.target.value));
    },
    [dispatch]
  );
  const handleAmountChange = useCallback(
    (event) => {
      dispatch(setAmount(event.target.value));
    },
    [dispatch]
  );
  const handleNextClick = useCallback(() => {
    dispatch(incrementStep());
  }, [dispatch]);
  return (
    <>
      <StepDescription>
        <div style={{ display: "flex", alignItems: "center" }}>
          Select tokens to send through the Wormhole Token Bridge.
          <div style={{ flexGrow: 1 }} />
          <Button
            onClick={() => setIsRecoveryOpen(true)}
            size="small"
            variant="outlined"
            endIcon={<Restore />}
          >
            Perform Recovery
          </Button>
        </div>
      </StepDescription>
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
      {isReady || uiAmountString ? (
        <div className={classes.transferField}>
          <TokenSelector disabled={shouldLockFields} />
        </div>
      ) : null}
      <TextField
        label="Amount"
        type="number"
        fullWidth
        className={classes.transferField}
        value={amount}
        onChange={handleAmountChange}
        disabled={shouldLockFields}
      />
      <ButtonWithLoader
        disabled={!isSourceComplete}
        onClick={handleNextClick}
        showLoader={false}
        error={statusMessage || error}
      >
        Next
      </ButtonWithLoader>
    </>
  );
}

export default Source;
