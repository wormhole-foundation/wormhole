import { ChainId } from "@certusone/wormhole-sdk";
import {
  Container,
  Step,
  StepButton,
  StepContent,
  Stepper,
} from "@material-ui/core";
import { useEffect, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useLocation } from "react-router";
import useCheckIfWormholeWrapped from "../../hooks/useCheckIfWormholeWrapped";
import useFetchTargetAsset from "../../hooks/useFetchTargetAsset";
import {
  selectTransferActiveStep,
  selectTransferIsRedeemComplete,
  selectTransferIsRedeeming,
  selectTransferIsSendComplete,
  selectTransferIsSending,
} from "../../store/selectors";
import {
  setSourceChain,
  setStep,
  setTargetChain,
} from "../../store/transferSlice";
import { CHAINS_BY_ID } from "../../utils/consts";
import Redeem from "./Redeem";
import RedeemPreview from "./RedeemPreview";
import Send from "./Send";
import SendPreview from "./SendPreview";
import Source from "./Source";
import SourcePreview from "./SourcePreview";
import Target from "./Target";
import TargetPreview from "./TargetPreview";

function Transfer() {
  useCheckIfWormholeWrapped();
  useFetchTargetAsset();
  const dispatch = useDispatch();
  const activeStep = useSelector(selectTransferActiveStep);
  const isSending = useSelector(selectTransferIsSending);
  const isSendComplete = useSelector(selectTransferIsSendComplete);
  const isRedeeming = useSelector(selectTransferIsRedeeming);
  const isRedeemComplete = useSelector(selectTransferIsRedeemComplete);
  const preventNavigation =
    (isSending || isSendComplete || isRedeeming) && !isRedeemComplete;

  const { search } = useLocation();
  const query = useMemo(() => new URLSearchParams(search), [search]);
  const pathSourceChain = query.get("sourceChain");
  const pathTargetChain = query.get("targetChain");

  //This effect initializes the state based on the path params
  useEffect(() => {
    if (!pathSourceChain && !pathTargetChain) {
      return;
    }
    try {
      const sourceChain: ChainId =
        CHAINS_BY_ID[parseFloat(pathSourceChain || "") as ChainId]?.id;
      const targetChain: ChainId =
        CHAINS_BY_ID[parseFloat(pathTargetChain || "") as ChainId]?.id;

      if (sourceChain === targetChain) {
        return;
      }
      if (sourceChain) {
        dispatch(setSourceChain(sourceChain));
      }
      if (targetChain) {
        dispatch(setTargetChain(targetChain));
      }
    } catch (e) {
      console.error("Invalid path params specified.");
    }
  }, [pathSourceChain, pathTargetChain, dispatch]);

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
      <Stepper activeStep={activeStep} orientation="vertical">
        <Step
          expanded={activeStep >= 0}
          disabled={preventNavigation || isRedeemComplete}
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
          disabled={preventNavigation || isRedeemComplete}
        >
          <StepButton
            disabled={preventNavigation || isRedeemComplete || activeStep === 0}
            onClick={() => dispatch(setStep(1))}
            icon={null}
          >
            2. Target
          </StepButton>
          <StepContent>
            {activeStep === 1 ? <Target /> : <TargetPreview />}
          </StepContent>
        </Step>
        <Step expanded={activeStep >= 2} disabled={isSendComplete}>
          <StepButton disabled icon={null}>
            3. Send tokens
          </StepButton>
          <StepContent>
            {activeStep === 2 ? <Send /> : <SendPreview />}
          </StepContent>
        </Step>
        <Step expanded={activeStep >= 3} completed={isRedeemComplete}>
          <StepButton
            onClick={() => dispatch(setStep(3))}
            disabled={!isSendComplete || isRedeemComplete}
            icon={null}
          >
            4. Redeem tokens
          </StepButton>
          <StepContent>
            {isRedeemComplete ? <RedeemPreview /> : <Redeem />}
          </StepContent>
        </Step>
      </Stepper>
    </Container>
  );
}

export default Transfer;
