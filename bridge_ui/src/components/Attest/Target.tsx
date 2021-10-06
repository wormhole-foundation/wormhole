import { makeStyles, MenuItem, TextField, Typography } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useBetaContext } from "../../contexts/BetaContext";
import { EthGasEstimateSummary } from "../../hooks/useTransactionFees";
import { incrementStep, setTargetChain } from "../../store/attestSlice";
import {
  selectAttestIsTargetComplete,
  selectAttestShouldLockFields,
  selectAttestSourceChain,
  selectAttestTargetChain,
} from "../../store/selectors";
import { BETA_CHAINS, CHAINS, CHAINS_BY_ID } from "../../utils/consts";
import { isEVMChain } from "../../utils/ethereum";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import LowBalanceWarning from "../LowBalanceWarning";

const useStyles = makeStyles((theme) => ({
  alert: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
}));

function Target() {
  const classes = useStyles();
  const dispatch = useDispatch();
  const isBeta = useBetaContext();
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
  const handleNextClick = useCallback(() => {
    dispatch(incrementStep());
  }, [dispatch]);
  return (
    <>
      <TextField
        select
        variant="outlined"
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
      <KeyAndBalance chainId={targetChain} />
      <Alert severity="info" className={classes.alert}>
        <Typography>
          You will have to pay transaction fees on{" "}
          {CHAINS_BY_ID[targetChain].name} to attest this token.{" "}
        </Typography>
        {isEVMChain(targetChain) && (
          <EthGasEstimateSummary
            methodType="createWrapped"
            chainId={targetChain}
          />
        )}
      </Alert>
      <LowBalanceWarning chainId={targetChain} />
      <ButtonWithLoader
        disabled={!isTargetComplete}
        onClick={handleNextClick}
        showLoader={false}
      >
        Next
      </ButtonWithLoader>
    </>
  );
}

export default Target;
