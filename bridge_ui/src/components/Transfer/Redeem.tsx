import {
  CHAIN_ID_AURORA,
  CHAIN_ID_AVAX,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_ETHEREUM_ROPSTEN,
  CHAIN_ID_FANTOM,
  CHAIN_ID_OASIS,
  CHAIN_ID_POLYGON,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  isEVMChain,
  WSOL_ADDRESS,
} from "@certusone/wormhole-sdk";
import {
  Button,
  Checkbox,
  CircularProgress,
  FormControlLabel,
  Link,
  makeStyles,
  Tooltip,
  Typography,
} from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { useCallback, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import useGetIsTransferCompleted from "../../hooks/useGetIsTransferCompleted";
import { useHandleRedeem } from "../../hooks/useHandleRedeem";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import {
  selectTransferIsRecovery,
  selectTransferTargetAsset,
  selectTransferTargetChain,
  selectTransferUseRelayer,
} from "../../store/selectors";
import { reset } from "../../store/transferSlice";
import {
  CLUSTER,
  getHowToAddTokensToWalletUrl,
  ROPSTEN_WETH_ADDRESS,
  WAVAX_ADDRESS,
  WBNB_ADDRESS,
  WETH_ADDRESS,
  WETH_AURORA_ADDRESS,
  WFTM_ADDRESS,
  WMATIC_ADDRESS,
  WROSE_ADDRESS,
} from "../../utils/consts";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import SmartAddress from "../SmartAddress";
import { SolanaCreateAssociatedAddressAlternate } from "../SolanaCreateAssociatedAddress";
import SolanaTPSWarning from "../SolanaTPSWarning";
import StepDescription from "../StepDescription";
import TerraFeeDenomPicker from "../TerraFeeDenomPicker";
import AddToMetamask from "./AddToMetamask";
import RedeemPreview from "./RedeemPreview";
import WaitingForWalletMessage from "./WaitingForWalletMessage";

const useStyles = makeStyles((theme) => ({
  alert: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
  centered: {
    margin: theme.spacing(4, 0, 2),
    textAlign: "center",
  },
}));

function Redeem() {
  const { handleClick, handleNativeClick, disabled, showLoader } =
    useHandleRedeem();
  const useRelayer = useSelector(selectTransferUseRelayer);
  const [manualRedeem, setManualRedeem] = useState(!useRelayer);
  const handleManuallyRedeemClick = useCallback(() => {
    setManualRedeem(true);
  }, []);
  const targetChain = useSelector(selectTransferTargetChain);
  const targetAsset = useSelector(selectTransferTargetAsset);
  const isRecovery = useSelector(selectTransferIsRecovery);
  const { isTransferCompletedLoading, isTransferCompleted } =
    useGetIsTransferCompleted(
      useRelayer ? false : true,
      useRelayer ? 5000 : undefined
    );
  const classes = useStyles();
  const dispatch = useDispatch();
  const { isReady, statusMessage } = useIsWalletReady(targetChain);
  //TODO better check, probably involving a hook & the VAA
  const isEthNative =
    targetChain === CHAIN_ID_ETH &&
    targetAsset &&
    targetAsset.toLowerCase() === WETH_ADDRESS.toLowerCase();
  const isEthRopstenNative =
    targetChain === CHAIN_ID_ETHEREUM_ROPSTEN &&
    targetAsset &&
    targetAsset.toLowerCase() === ROPSTEN_WETH_ADDRESS.toLowerCase();
  const isBscNative =
    targetChain === CHAIN_ID_BSC &&
    targetAsset &&
    targetAsset.toLowerCase() === WBNB_ADDRESS.toLowerCase();
  const isPolygonNative =
    targetChain === CHAIN_ID_POLYGON &&
    targetAsset &&
    targetAsset.toLowerCase() === WMATIC_ADDRESS.toLowerCase();
  const isAvaxNative =
    targetChain === CHAIN_ID_AVAX &&
    targetAsset &&
    targetAsset.toLowerCase() === WAVAX_ADDRESS.toLowerCase();
  const isOasisNative =
    targetChain === CHAIN_ID_OASIS &&
    targetAsset &&
    targetAsset.toLowerCase() === WROSE_ADDRESS.toLowerCase();
  const isAuroraNative =
    targetChain === CHAIN_ID_AURORA &&
    targetAsset &&
    targetAsset.toLowerCase() === WETH_AURORA_ADDRESS.toLowerCase();
  const isFantomNative =
    targetChain === CHAIN_ID_FANTOM &&
    targetAsset &&
    targetAsset.toLowerCase() === WFTM_ADDRESS.toLowerCase();
  const isSolNative =
    targetChain === CHAIN_ID_SOLANA &&
    targetAsset &&
    targetAsset === WSOL_ADDRESS;
  const isNativeEligible =
    isEthNative ||
    isEthRopstenNative ||
    isBscNative ||
    isPolygonNative ||
    isAvaxNative ||
    isOasisNative ||
    isAuroraNative ||
    isFantomNative ||
    isSolNative;
  const [useNativeRedeem, setUseNativeRedeem] = useState(true);
  const toggleNativeRedeem = useCallback(() => {
    setUseNativeRedeem(!useNativeRedeem);
  }, [useNativeRedeem]);
  const handleResetClick = useCallback(() => {
    dispatch(reset());
  }, [dispatch]);
  const howToAddTokensUrl = getHowToAddTokensToWalletUrl(targetChain);

  const relayerContent = (
    <>
      {isEVMChain(targetChain) && !isTransferCompleted ? (
        <KeyAndBalance chainId={targetChain} />
      ) : null}

      {!isReady && isEVMChain(targetChain) && !isTransferCompleted ? (
        <Typography className={classes.centered}>
          {"Please connect your wallet to check for transfer completion."}
        </Typography>
      ) : null}

      {(!isEVMChain(targetChain) || isReady) && !isTransferCompleted ? (
        <div className={classes.centered}>
          <CircularProgress style={{ marginBottom: 16 }} />
          <Typography>
            {"Waiting for a relayer to process your transfer."}
          </Typography>
          <Tooltip title="Your fees will be refunded on the target chain">
            <Button
              onClick={handleManuallyRedeemClick}
              size="small"
              variant="outlined"
              style={{ marginTop: 16 }}
            >
              Manually redeem instead
            </Button>
          </Tooltip>
        </div>
      ) : null}

      {isTransferCompleted ? (
        <RedeemPreview overrideExplainerString="Success! Your transfer is complete." />
      ) : null}
    </>
  );

  const nonRelayContent = (
    <>
      <KeyAndBalance chainId={targetChain} />
      {targetChain === CHAIN_ID_TERRA && (
        <TerraFeeDenomPicker disabled={disabled} />
      )}
      {isNativeEligible && (
        <FormControlLabel
          control={
            <Checkbox
              checked={useNativeRedeem}
              onChange={toggleNativeRedeem}
              color="primary"
            />
          }
          label="Automatically unwrap to native currency"
        />
      )}
      {targetChain === CHAIN_ID_SOLANA && CLUSTER === "mainnet" && (
        <SolanaTPSWarning />
      )}
      {targetChain === CHAIN_ID_SOLANA ? (
        <SolanaCreateAssociatedAddressAlternate />
      ) : null}

      <>
        {" "}
        <ButtonWithLoader
          //TODO disable when the associated token account is confirmed to not exist
          disabled={
            !isReady ||
            disabled ||
            (isRecovery && (isTransferCompletedLoading || isTransferCompleted))
          }
          onClick={
            isNativeEligible && useNativeRedeem
              ? handleNativeClick
              : handleClick
          }
          showLoader={showLoader || (isRecovery && isTransferCompletedLoading)}
          error={statusMessage}
        >
          Redeem
        </ButtonWithLoader>
        <WaitingForWalletMessage />
      </>

      {isRecovery && isReady && isTransferCompleted ? (
        <>
          <Alert severity="info" variant="outlined" className={classes.alert}>
            These tokens have already been redeemed.{" "}
            {!isEVMChain(targetChain) && howToAddTokensUrl ? (
              <Link
                href={howToAddTokensUrl}
                target="_blank"
                rel="noopener noreferrer"
              >
                Click here to see how to add them to your wallet.
              </Link>
            ) : null}
          </Alert>
          {targetAsset ? (
            <>
              <span>Token Address:</span>
              <SmartAddress
                chainId={targetChain}
                address={targetAsset || undefined}
              />
            </>
          ) : null}
          {isEVMChain(targetChain) ? <AddToMetamask /> : null}
          <ButtonWithLoader onClick={handleResetClick}>
            Transfer More Tokens!
          </ButtonWithLoader>
        </>
      ) : null}
    </>
  );

  return (
    <>
      <StepDescription>Receive the tokens on the target chain</StepDescription>
      {manualRedeem ? nonRelayContent : relayerContent}
    </>
  );
}

export default Redeem;
