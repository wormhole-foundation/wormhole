import {
  Container,
  makeStyles,
  Step,
  StepButton,
  StepContent,
  Stepper,
} from "@material-ui/core";
import { useEffect, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import useCheckIfWormholeWrapped from "../../hooks/useCheckIfWormholeWrapped";
import useFetchTargetAsset from "../../hooks/useFetchTargetAsset";
import useGetBalanceEffect from "../../hooks/useGetBalanceEffect";
import {
  selectTransferActiveStep,
  selectTransferIsRedeeming,
  selectTransferIsSendComplete,
  selectTransferIsSending,
} from "../../store/selectors";
import { setStep } from "../../store/transferSlice";
import Recovery from "./Recovery";
import Redeem from "./Redeem";
import Send from "./Send";
import Source from "./Source";
import Target from "./Target";

// TODO: ensure that both wallets are connected to the same known network
// TODO: loaders and such, navigation block?
// TODO: refresh displayed token amount after transfer somehow, could be resolved by having different components appear
// TODO: warn if amount exceeds balance

const useStyles = makeStyles(() => ({
  rootContainer: {
    backgroundColor: "rgba(0,0,0,0.2)",
  },
}));

function Transfer() {
  const classes = useStyles();
  useCheckIfWormholeWrapped();
  useFetchTargetAsset();
  useGetBalanceEffect("target");
  const [isRecoveryOpen, setIsRecoveryOpen] = useState(false);
  const dispatch = useDispatch();
  const activeStep = useSelector(selectTransferActiveStep);
  const isSending = useSelector(selectTransferIsSending);
  const isSendComplete = useSelector(selectTransferIsSendComplete);
  const isRedeeming = useSelector(selectTransferIsRedeeming);
  const preventNavigation = isSending || isSendComplete || isRedeeming;
  useEffect(() => {
    if (preventNavigation) {
      window.onbeforeunload = () => true;
      return () => {
        window.onbeforeunload = null;
      };
    }
  }, [preventNavigation]);
  return (
    <Container maxWidth="md">
      <Stepper
        activeStep={activeStep}
        orientation="vertical"
        className={classes.rootContainer}
      >
        <Step>
          <StepButton onClick={() => dispatch(setStep(0))}>Source</StepButton>
          <StepContent>
            <Source setIsRecoveryOpen={setIsRecoveryOpen} />
          </StepContent>
        </Step>
        <Step>
          <StepButton onClick={() => dispatch(setStep(1))}>Target</StepButton>
          <StepContent>
            <Target />
          </StepContent>
        </Step>
        <Step>
          <StepButton onClick={() => dispatch(setStep(2))}>
            Send tokens
          </StepButton>
          <StepContent>
            <Send />
          </StepContent>
        </Step>
        <Step>
          <StepButton
            onClick={() => dispatch(setStep(3))}
            disabled={!isSendComplete}
          >
            Redeem tokens
          </StepButton>
          <StepContent>
            <Redeem />
          </StepContent>
        </Step>
      </Stepper>
      <Recovery open={isRecoveryOpen} setOpen={setIsRecoveryOpen} />
    </Container>
  );
}

export default Transfer;
