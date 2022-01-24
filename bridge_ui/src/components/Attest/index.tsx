import {
  Container,
  Step,
  StepButton,
  StepContent,
  Stepper,
} from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import { setStep } from "../../store/attestSlice";
import {
  selectAttestActiveStep,
  selectAttestIsCreateComplete,
  selectAttestIsCreating,
  selectAttestIsSendComplete,
  selectAttestIsSending,
} from "../../store/selectors";
import HeaderText from "../HeaderText";
import Create from "./Create";
import CreatePreview from "./CreatePreview";
import Send from "./Send";
import SendPreview from "./SendPreview";
import Source from "./Source";
import SourcePreview from "./SourcePreview";
import Target from "./Target";
import TargetPreview from "./TargetPreview";

function Attest() {
  const dispatch = useDispatch();
  const activeStep = useSelector(selectAttestActiveStep);
  const isSending = useSelector(selectAttestIsSending);
  const isSendComplete = useSelector(selectAttestIsSendComplete);
  const isCreating = useSelector(selectAttestIsCreating);
  const isCreateComplete = useSelector(selectAttestIsCreateComplete);
  const preventNavigation =
    (isSending || isSendComplete || isCreating) && !isCreateComplete;
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
      <HeaderText white>Token Registration</HeaderText>
      <Alert severity="info">
        This form allows you to register a token on a new foreign chain. Tokens
        must be registered before they can be transferred.
      </Alert>
      <Stepper activeStep={activeStep} orientation="vertical">
        <Step
          expanded={activeStep >= 0}
          disabled={preventNavigation || isCreateComplete}
        >
          <StepButton onClick={() => dispatch(setStep(0))} icon={null}>
            1. Source
          </StepButton>
          <StepContent>
            {activeStep === 0 ? <Source /> : <SourcePreview />}
          </StepContent>
        </Step>
        <Step
          expanded={activeStep >= 1}
          disabled={preventNavigation || isCreateComplete}
        >
          <StepButton onClick={() => dispatch(setStep(1))} icon={null}>
            2. Target
          </StepButton>
          <StepContent>
            {activeStep === 1 ? <Target /> : <TargetPreview />}
          </StepContent>
        </Step>
        <Step expanded={activeStep >= 2} disabled={isSendComplete}>
          <StepButton onClick={() => dispatch(setStep(2))} icon={null}>
            3. Send attestation
          </StepButton>
          <StepContent>
            {activeStep === 2 ? <Send /> : <SendPreview />}
          </StepContent>
        </Step>
        <Step expanded={activeStep >= 3}>
          <StepButton
            onClick={() => dispatch(setStep(3))}
            disabled={!isSendComplete}
            icon={null}
          >
            4. Create wrapped token
          </StepButton>
          <StepContent>
            {isCreateComplete ? <CreatePreview /> : <Create />}
          </StepContent>
        </Step>
      </Stepper>
    </Container>
  );
}

export default Attest;
