import {
  CHAIN_ID_TERRA,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import { Button, CircularProgress, makeStyles } from "@material-ui/core";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useConnectedWallet } from "@terra-money/wallet-provider";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../../contexts/SolanaWalletContext";
import useTransferSignedVAA from "../../hooks/useTransferSignedVAA";
import {
  selectTransferIsRedeeming,
  selectTransferIsSourceAssetWormholeWrapped,
  selectTransferOriginChain,
  selectTransferTargetAsset,
  selectTransferTargetChain,
} from "../../store/selectors";
import { reset, setIsRedeeming } from "../../store/transferSlice";
import {
  redeemOnEth,
  redeemOnSolana,
  redeemOnTerra,
} from "../../utils/redeemOn";

const useStyles = makeStyles((theme) => ({
  transferButton: {
    marginTop: theme.spacing(2),
    textTransform: "none",
    width: "100%",
  },
}));

function Redeem() {
  const dispatch = useDispatch();
  const classes = useStyles();
  const isSourceAssetWormholeWrapped = useSelector(
    selectTransferIsSourceAssetWormholeWrapped
  );
  const originChain = useSelector(selectTransferOriginChain);
  const targetChain = useSelector(selectTransferTargetChain);
  const targetAsset = useSelector(selectTransferTargetAsset);
  const { wallet } = useSolanaWallet();
  const solPK = wallet?.publicKey;
  const { signer } = useEthereumProvider();
  const terraWallet = useConnectedWallet();
  const signedVAA = useTransferSignedVAA();
  const isRedeeming = useSelector(selectTransferIsRedeeming);
  const handleRedeemClick = useCallback(() => {
    if (targetChain === CHAIN_ID_ETH && signedVAA) {
      (async () => {
        dispatch(setIsRedeeming(true));
        await redeemOnEth(signer, signedVAA);
        dispatch(reset());
      })();
    }
    if (targetChain === CHAIN_ID_SOLANA && signedVAA) {
      (async () => {
        dispatch(setIsRedeeming(true));
        await redeemOnSolana(
          wallet,
          solPK?.toString(),
          signedVAA,
          !!isSourceAssetWormholeWrapped && originChain === CHAIN_ID_SOLANA,
          targetAsset || undefined
        );
        dispatch(reset());
      })();
    }
    if (targetChain === CHAIN_ID_TERRA && signedVAA) {
      dispatch(setIsRedeeming(true));
      redeemOnTerra(terraWallet, signedVAA);
    }
  }, [
    dispatch,
    terraWallet,
    targetChain,
    signer,
    signedVAA,
    wallet,
    solPK,
    isSourceAssetWormholeWrapped,
    originChain,
    targetAsset,
  ]);
  return (
    <div style={{ position: "relative" }}>
      <Button
        color="primary"
        variant="contained"
        className={classes.transferButton}
        disabled={isRedeeming}
        onClick={handleRedeemClick}
      >
        Redeem
      </Button>
      {isRedeeming ? (
        <CircularProgress
          size={24}
          color="inherit"
          style={{
            position: "absolute",
            bottom: 0,
            left: "50%",
            marginLeft: -12,
            marginBottom: 6,
          }}
        />
      ) : null}
    </div>
  );
}

export default Redeem;
