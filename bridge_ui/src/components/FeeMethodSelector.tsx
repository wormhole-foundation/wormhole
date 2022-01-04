import { CHAIN_ID_TERRA, isEVMChain } from "@certusone/wormhole-sdk";
import { Card, Checkbox, makeStyles, Typography } from "@material-ui/core";
import { useCallback, useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import SmartAddress from "../components/SmartAddress";
import useRelayerInfo from "../hooks/useRelayerInfo";
import { GasEstimateSummary } from "../hooks/useTransactionFees";
import { COLORS } from "../muiTheme";
import {
  selectTransferOriginAsset,
  selectTransferOriginChain,
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccount,
  selectTransferTargetChain,
  selectTransferUseRelayer,
} from "../store/selectors";
import { setRelayerFee, setUseRelayer } from "../store/transferSlice";
import { getDefaultNativeCurrencySymbol } from "../utils/consts";

const useStyles = makeStyles((theme) => ({
  feeSelectorContainer: {
    marginTop: "2rem",
    textAlign: "center",
  },
  title: {
    margin: theme.spacing(2),
  },
  optionCardBase: {
    display: "flex",
    margin: theme.spacing(2),
    alignItems: "center",
    justifyContent: "space-between",
    padding: theme.spacing(1),
    background: COLORS.nearBlackWithMinorTransparency,
    "& > *": {
      margin: ".5rem",
    },
    border: "1px solid " + COLORS.nearBlackWithMinorTransparency,
  },
  alignCenterContainer: {
    alignItems: "center",
    display: "flex",
    "& > *": {
      margin: "0rem 1rem 0rem 1rem",
    },
  },
  optionCardSelectable: {
    "&:hover": {
      cursor: "pointer",
      boxShadow: "inset 0 0 100px 100px rgba(255, 255, 255, 0.1)",
    },
  },
  optionCardSelected: {
    border: "1px solid " + COLORS.blue,
  },
  inlineBlock: {
    display: "inline-block",
  },
}));

function FeeMethodSelector() {
  const classes = useStyles();
  const originAsset = useSelector(selectTransferOriginAsset);
  const originChain = useSelector(selectTransferOriginChain);
  const targetChain = useSelector(selectTransferTargetChain);
  const relayerInfo = useRelayerInfo(originChain, originAsset, targetChain);
  const dispatch = useDispatch();
  const relayerSelected = !!useSelector(selectTransferUseRelayer);
  const sourceParsedTokenAccount = useSelector(
    selectTransferSourceParsedTokenAccount
  );
  const sourceSymbol = sourceParsedTokenAccount?.symbol;
  const sourceChain = useSelector(selectTransferSourceChain);

  console.log("relayer info in fee method selector", relayerInfo);

  const relayerEligible =
    relayerInfo.data &&
    relayerInfo.data.isRelayable &&
    relayerInfo.data.feeFormatted &&
    relayerInfo.data.feeUsd;

  const chooseRelayer = useCallback(() => {
    if (relayerEligible) {
      dispatch(setUseRelayer(true));
      dispatch(setRelayerFee(relayerInfo.data?.feeFormatted));
    }
  }, [relayerInfo, dispatch, relayerEligible]);

  const chooseManual = useCallback(() => {
    dispatch(setUseRelayer(false));
    dispatch(setRelayerFee(undefined));
  }, [dispatch]);

  useEffect(() => {
    if (relayerInfo.data?.isRelayable === true) {
      chooseRelayer();
    } else if (relayerInfo.data?.isRelayable === false) {
      chooseManual();
    }
    //If it's undefined / null it's still loading, so no action is taken.
  }, [relayerInfo, chooseRelayer, chooseManual]);

  const relayerContent = (
    <Card
      className={
        classes.optionCardBase +
        " " +
        (relayerSelected ? classes.optionCardSelected : "") +
        " " +
        (relayerEligible ? classes.optionCardSelectable : "")
      }
      onClick={chooseRelayer}
    >
      <div className={classes.alignCenterContainer}>
        <Checkbox
          checked={relayerSelected}
          disabled={!relayerEligible}
          onClick={chooseRelayer}
          className={classes.inlineBlock}
        />
        <div className={classes.inlineBlock}>
          {relayerEligible ? (
            <div>
              <Typography variant="body1">{"Automatic Payment"}</Typography>
              <Typography variant="body2" color="textSecondary">
                {"Use a relayer to pay with additional " +
                  (sourceSymbol
                    ? sourceSymbol + ""
                    : "the token you're transferring.")}{" "}
              </Typography>
            </div>
          ) : (
            <>
              <Typography color="textSecondary" variant="body2">
                {"Automatic redeem is unavailable for this token."}
              </Typography>
              <div />
            </>
          )}
        </div>
      </div>
      {/* TODO fixed number of decimals on these strings */}
      {relayerEligible ? (
        <div>
          <div>
            <Typography className={classes.inlineBlock}>
              {"~ " +
                parseFloat(relayerInfo.data?.feeFormatted || "0").toFixed(5)}
            </Typography>
            <SmartAddress
              chainId={sourceChain}
              parsedTokenAccount={sourceParsedTokenAccount}
            />
          </div>
          <Typography>{`($ ${relayerInfo.data?.feeUsd})`}</Typography>
        </div>
      ) : null}
    </Card>
  );

  const manualRedeemContent = (
    <Card
      className={
        classes.optionCardBase +
        " " +
        classes.optionCardSelectable +
        " " +
        (!relayerSelected ? classes.optionCardSelected : "")
      }
      onClick={chooseManual}
    >
      <div className={classes.alignCenterContainer}>
        <Checkbox
          checked={!relayerSelected}
          onClick={chooseManual}
          className={classes.inlineBlock}
        />
        <div className={classes.inlineBlock}>
          <Typography variant="body1">{"Manual Payment"}</Typography>
          <Typography variant="body2" color="textSecondary">
            {"Pay with your own " + getDefaultNativeCurrencySymbol(targetChain)}
          </Typography>
        </div>
      </div>
      {(isEVMChain(targetChain) || targetChain === CHAIN_ID_TERRA) && (
        <GasEstimateSummary
          methodType="transfer"
          chainId={targetChain}
          priceQuote={relayerInfo.data?.targetNativeAssetPriceQuote}
        />
      )}
    </Card>
  );

  return (
    <div className={classes.feeSelectorContainer}>
      <Typography
        className={classes.title}
        variant="subtitle2"
        color="textSecondary"
      >
        How would you like to pay the target chain fees?
      </Typography>
      {relayerContent}
      {manualRedeemContent}
    </div>
  );
}

export default FeeMethodSelector;
