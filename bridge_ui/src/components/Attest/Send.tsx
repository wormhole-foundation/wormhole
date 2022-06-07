import { CHAIN_ID_SOLANA, isTerraChain } from "@certusone/wormhole-sdk";
import { Alert } from "@material-ui/lab";
import { Link, makeStyles } from "@material-ui/core";
import { useMemo } from "react";
import { useSelector } from "react-redux";
import { useHandleAttest } from "../../hooks/useHandleAttest";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import useMetaplexData from "../../hooks/useMetaplexData";
import {
  selectAttestAttestTx,
  selectAttestIsSendComplete,
  selectAttestSourceAsset,
  selectAttestSourceChain,
} from "../../store/selectors";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import TransactionProgress from "../TransactionProgress";
import WaitingForWalletMessage from "./WaitingForWalletMessage";
import { SOLANA_TOKEN_METADATA_PROGRAM_URL } from "../../utils/consts";
import TerraFeeDenomPicker from "../TerraFeeDenomPicker";

const useStyles = makeStyles((theme) => ({
  alert: {
    marginTop: theme.spacing(1),
  },
}));

const SolanaTokenMetadataWarning = () => {
  const sourceAsset = useSelector(selectAttestSourceAsset);
  const sourceAssetArrayed = useMemo(() => {
    return [sourceAsset];
  }, [sourceAsset]);
  const metaplexData = useMetaplexData(sourceAssetArrayed);
  const classes = useStyles();

  if (metaplexData.isFetching || metaplexData.error) {
    return null;
  }

  return !metaplexData.data?.get(sourceAsset) ? (
    <Alert severity="warning" variant="outlined" className={classes.alert}>
      This token is missing on-chain (Metaplex) metadata. Without it, the
      wrapped token's name and symbol will be empty. See the{" "}
      <Link
        href={SOLANA_TOKEN_METADATA_PROGRAM_URL}
        target="_blank"
        rel="noopener noreferrer"
      >
        metaplex repository
      </Link>{" "}
      for details.
    </Alert>
  ) : null;
};

function Send() {
  const { handleClick, disabled, showLoader } = useHandleAttest();
  const sourceChain = useSelector(selectAttestSourceChain);
  const attestTx = useSelector(selectAttestAttestTx);
  const isSendComplete = useSelector(selectAttestIsSendComplete);
  const { isReady, statusMessage } = useIsWalletReady(sourceChain);

  return (
    <>
      <KeyAndBalance chainId={sourceChain} />
      {isTerraChain(sourceChain) && (
        <TerraFeeDenomPicker disabled={disabled} chainId={sourceChain} />
      )}
      <ButtonWithLoader
        disabled={!isReady || disabled}
        onClick={handleClick}
        showLoader={showLoader}
        error={statusMessage}
      >
        Attest
      </ButtonWithLoader>
      {sourceChain === CHAIN_ID_SOLANA && <SolanaTokenMetadataWarning />}
      <WaitingForWalletMessage />
      <TransactionProgress
        chainId={sourceChain}
        tx={attestTx}
        isSendComplete={isSendComplete}
      />
    </>
  );
}

export default Send;
