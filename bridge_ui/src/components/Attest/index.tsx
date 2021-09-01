import {
  Container,
  Step,
  StepButton,
  StepContent,
  Stepper,
} from "@material-ui/core";
import { useDispatch, useSelector } from "react-redux";
import {
  selectAttestActiveStep,
  selectAttestSignedVAAHex,
} from "../../store/selectors";
import { setStep } from "../../store/attestSlice";
import Create from "./Create";
import Send from "./Send";
import Source from "./Source";
import Target from "./Target";
import { Alert } from "@material-ui/lab";

// TODO: ensure that both wallets are connected to the same known network

function Attest() {
  const dispatch = useDispatch();
  const activeStep = useSelector(selectAttestActiveStep);
  const signedVAAHex = useSelector(selectAttestSignedVAAHex);
  return (
    <Container maxWidth="md">
      <Alert severity="info">
        This form allows you to register a token on a new foreign chain. Tokens
        must be registered before they can be transferred.
      </Alert>
      <Stepper activeStep={activeStep} orientation="vertical">
        <Step>
          <StepButton onClick={() => dispatch(setStep(0))}>
            Select a source
          </StepButton>
          <StepContent>
            <Source />
          </StepContent>
        </Step>
        <Step>
          <StepButton onClick={() => dispatch(setStep(1))}>
            Select a target
          </StepButton>
          <StepContent>
            <Target />
          </StepContent>
        </Step>
        <Step>
          <StepButton onClick={() => dispatch(setStep(2))}>
            Send attestation
          </StepButton>
          <StepContent>
            <Send />
          </StepContent>
        </Step>
        <Step>
          <StepButton
            onClick={() => dispatch(setStep(3))}
            disabled={!signedVAAHex}
          >
            Create wrapped token
          </StepButton>
          <StepContent>
            <Create />
          </StepContent>
        </Step>
      </Stepper>
    </Container>
  );
}

export default Attest;
