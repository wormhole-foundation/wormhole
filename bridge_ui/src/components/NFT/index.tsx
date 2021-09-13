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
import {
  selectNFTActiveStep,
  selectNFTIsRedeemComplete,
  selectNFTIsRedeeming,
  selectNFTIsSendComplete,
  selectNFTIsSending,
} from "../../store/selectors";
import { setStep } from "../../store/nftSlice";
import Recovery from "./Recovery";
import Redeem from "./Redeem";
import RedeemPreview from "./RedeemPreview";
import Send from "./Send";
import SendPreview from "./SendPreview";
import Source from "./Source";
import SourcePreview from "./SourcePreview";
import Target from "./Target";
import TargetPreview from "./TargetPreview";

const useStyles = makeStyles(() => ({
  rootContainer: {
    backgroundColor: "rgba(0,0,0,0.2)",
  },
}));

function NFT() {
  const classes = useStyles();
  useCheckIfWormholeWrapped(true);
  useFetchTargetAsset(true);
  const [isRecoveryOpen, setIsRecoveryOpen] = useState(false);
  const dispatch = useDispatch();
  const activeStep = useSelector(selectNFTActiveStep);
  const isSending = useSelector(selectNFTIsSending);
  const isSendComplete = useSelector(selectNFTIsSendComplete);
  const isRedeeming = useSelector(selectNFTIsRedeeming);
  const isRedeemComplete = useSelector(selectNFTIsRedeemComplete);
  const preventNavigation =
    (isSending || isSendComplete || isRedeeming) && !isRedeemComplete;
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
        <Step
          expanded={activeStep >= 0}
          disabled={preventNavigation || isRedeemComplete}
        >
          <StepButton onClick={() => dispatch(setStep(0))}>Source</StepButton>
          <StepContent>
            {activeStep === 0 ? (
              <Source setIsRecoveryOpen={setIsRecoveryOpen} />
            ) : (
              <SourcePreview />
            )}
          </StepContent>
        </Step>
        <Step
          expanded={activeStep >= 1}
          disabled={preventNavigation || isRedeemComplete}
        >
          <StepButton onClick={() => dispatch(setStep(1))}>Target</StepButton>
          <StepContent>
            {activeStep === 1 ? <Target /> : <TargetPreview />}
          </StepContent>
        </Step>
        <Step expanded={activeStep >= 2} disabled={isSendComplete}>
          <StepButton onClick={() => dispatch(setStep(2))}>Send NFT</StepButton>
          <StepContent>
            {activeStep === 2 ? <Send /> : <SendPreview />}
          </StepContent>
        </Step>
        <Step expanded={activeStep >= 3}>
          <StepButton
            onClick={() => dispatch(setStep(3))}
            disabled={!isSendComplete}
          >
            Redeem NFT
          </StepButton>
          <StepContent>
            {isRedeemComplete ? <RedeemPreview /> : <Redeem />}
          </StepContent>
        </Step>
      </Stepper>
      <Recovery
        open={isRecoveryOpen}
        setOpen={setIsRecoveryOpen}
        disabled={preventNavigation}
      />
    </Container>
  );
}

export default NFT;
