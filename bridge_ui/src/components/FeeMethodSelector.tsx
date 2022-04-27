import {
  CHAIN_ID_ACALA,
  CHAIN_ID_KARURA,
  CHAIN_ID_TERRA,
  hexToNativeString,
  isEVMChain,
} from "@certusone/wormhole-sdk";
import {
  Card,
  Checkbox,
  Chip,
  makeStyles,
  Typography,
} from "@material-ui/core";
import clsx from "clsx";
import { parseUnits } from "ethers/lib/utils";
import { useCallback, useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import SmartAddress from "../components/SmartAddress";
import { useAcalaRelayerInfo } from "../hooks/useAcalaRelayerInfo";
import useRelayerInfo from "../hooks/useRelayerInfo";
import { GasEstimateSummary } from "../hooks/useTransactionFees";
import { COLORS } from "../muiTheme";
import {
  selectTransferAmount,
  selectTransferOriginAsset,
  selectTransferOriginChain,
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccount,
  selectTransferTargetChain,
  selectTransferUseRelayer,
} from "../store/selectors";
import { setRelayerFee, setUseRelayer } from "../store/transferSlice";
import { CHAINS_BY_ID, getDefaultNativeCurrencySymbol } from "../utils/consts";

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
  alignLeft: {
    textAlign: "left",
  },
  betaLabel: {
    color: COLORS.white,
    background: "linear-gradient(20deg, #f44b1b 0%, #eeb430 100%)",
    marginLeft: theme.spacing(1),
    fontSize: "120%",
  },
}));

function FeeMethodSelector() {
  const classes = useStyles();
  const originAsset = useSelector(selectTransferOriginAsset);
  const originChain = useSelector(selectTransferOriginChain);
  const targetChain = useSelector(selectTransferTargetChain);
  const transferAmount = useSelector(selectTransferAmount);
  const relayerInfo = useRelayerInfo(originChain, originAsset, targetChain);
  const sourceParsedTokenAccount = useSelector(
    selectTransferSourceParsedTokenAccount
  );
  const sourceDecimals = sourceParsedTokenAccount?.decimals;
  let vaaNormalizedAmount: string | undefined = undefined;
  if (transferAmount && sourceDecimals !== undefined) {
    try {
      vaaNormalizedAmount = parseUnits(
        transferAmount,
        Math.min(sourceDecimals, 8)
      ).toString();
    } catch (e) {}
  }
  const sourceSymbol = sourceParsedTokenAccount?.symbol;
  const acalaRelayerInfo = useAcalaRelayerInfo(
    targetChain,
    vaaNormalizedAmount,
    originChain ? hexToNativeString(originAsset, originChain) : undefined
  );
  const sourceChain = useSelector(selectTransferSourceChain);
  const dispatch = useDispatch();
  const relayerSelected = !!useSelector(selectTransferUseRelayer);

  console.log("relayer info in fee method selector", relayerInfo);

  const relayerEligible =
    relayerInfo.data &&
    relayerInfo.data.isRelayable &&
    relayerInfo.data.feeFormatted &&
    relayerInfo.data.feeUsd;

  const targetIsAcala =
    targetChain === CHAIN_ID_ACALA || targetChain === CHAIN_ID_KARURA;
  const acalaRelayerEligible = acalaRelayerInfo.data?.shouldRelay;

  const chooseAcalaRelayer = useCallback(() => {
    if (targetIsAcala && acalaRelayerEligible) {
      dispatch(setUseRelayer(true));
      dispatch(setRelayerFee(undefined));
    }
  }, [dispatch, targetIsAcala, acalaRelayerEligible]);

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
    if (targetIsAcala) {
      if (acalaRelayerEligible) {
        chooseAcalaRelayer();
      } else {
        chooseManual();
      }
    } else if (relayerInfo.data?.isRelayable === true) {
      chooseRelayer();
    } else if (relayerInfo.data?.isRelayable === false) {
      chooseManual();
    }
    //If it's undefined / null it's still loading, so no action is taken.
  }, [
    relayerInfo,
    chooseRelayer,
    chooseManual,
    targetIsAcala,
    acalaRelayerEligible,
    chooseAcalaRelayer,
  ]);

  const acalaRelayerContent = (
    <Card
      className={
        classes.optionCardBase +
        " " +
        (relayerSelected ? classes.optionCardSelected : "") +
        " " +
        (acalaRelayerEligible ? classes.optionCardSelectable : "")
      }
      onClick={chooseAcalaRelayer}
    >
      <div className={classes.alignCenterContainer}>
        <Checkbox
          checked={relayerSelected}
          disabled={!acalaRelayerEligible}
          onClick={chooseAcalaRelayer}
          className={classes.inlineBlock}
        />
        <div className={clsx(classes.inlineBlock, classes.alignLeft)}>
          {acalaRelayerEligible ? (
            <div>
              <Typography variant="body1">
                {CHAINS_BY_ID[targetChain].name}
              </Typography>
              <Typography variant="body2" color="textSecondary">
                {CHAINS_BY_ID[targetChain].name} pays gas for you &#127881;
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
      {acalaRelayerEligible ? (
        <>
          <div></div>
          <div></div>
        </>
      ) : null}
    </Card>
  );

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
        <div className={clsx(classes.inlineBlock, classes.alignLeft)}>
          {relayerEligible ? (
            <div>
              <Typography variant="body1">Automatic Payment</Typography>
              <Typography variant="body2" color="textSecondary">
                {`Pay with additional ${
                  sourceSymbol ? sourceSymbol : "tokens"
                } and use a relayer`}
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
        <>
          <div>
            <Chip label="Beta" className={classes.betaLabel} />
          </div>
          <div>
            <div>
              <Typography className={classes.inlineBlock}>
                {/* Transfers are max 8 decimals */}
                {parseFloat(relayerInfo.data?.feeFormatted || "0").toFixed(
                  Math.min(sourceParsedTokenAccount?.decimals || 8, 8)
                )}
              </Typography>
              <SmartAddress
                chainId={sourceChain}
                parsedTokenAccount={sourceParsedTokenAccount}
              />
            </div>{" "}
            <Typography>{`($ ${relayerInfo.data?.feeUsd})`}</Typography>
          </div>
        </>
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
        <div className={clsx(classes.inlineBlock, classes.alignLeft)}>
          <Typography variant="body1">{"Manual Payment"}</Typography>
          <Typography variant="body2" color="textSecondary">
            {`Pay with your own ${
              targetChain === CHAIN_ID_TERRA
                ? "funds"
                : getDefaultNativeCurrencySymbol(targetChain)
            } on ${CHAINS_BY_ID[targetChain]?.name || "target chain"}`}
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
      {targetIsAcala ? acalaRelayerContent : relayerContent}
      {manualRedeemContent}
    </div>
  );
}

export default FeeMethodSelector;
