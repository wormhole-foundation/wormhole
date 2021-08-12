import {
  Container,
  Step,
  StepButton,
  StepContent,
  Stepper,
} from "@material-ui/core";
import { useDispatch, useSelector } from "react-redux";
import useCheckIfWormholeWrapped from "../../hooks/useCheckIfWormholeWrapped";
import useFetchTargetAsset from "../../hooks/useFetchTargetAsset";
import useGetBalanceEffect from "../../hooks/useGetBalanceEffect";
import {
  selectTransferActiveStep,
  selectTransferSignedVAAHex,
} from "../../store/selectors";
import { setStep } from "../../store/transferSlice";
import Redeem from "./Redeem";
import Send from "./Send";
import Source from "./Source";
import Target from "./Target";

// TODO: ensure that both wallets are connected to the same known network
// TODO: loaders and such, navigation block?
// TODO: refresh displayed token amount after transfer somehow, could be resolved by having different components appear
// TODO: warn if amount exceeds balance

function Transfer() {
  useGetBalanceEffect("source");
  useCheckIfWormholeWrapped();
  useFetchTargetAsset();
  useGetBalanceEffect("target");
  const dispatch = useDispatch();
  const activeStep = useSelector(selectTransferActiveStep);
  const signedVAAHex = useSelector(selectTransferSignedVAAHex);
  return (
    <Container maxWidth="md">
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
            Send tokens
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
            Redeem tokens
          </StepButton>
          <StepContent>
            <Redeem />
          </StepContent>
        </Step>
      </Stepper>
    </Container>
  );
}

export default Transfer;
