import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  postVaaSolana,
  redeemOnEth,
  redeemOnEthNative,
  redeemOnSolana,
  redeemOnTerra,
} from "@certusone/wormhole-sdk";
import { WalletContextState } from "@solana/wallet-adapter-react";
import { Connection } from "@solana/web3.js";
import {
  ConnectedWallet,
  useConnectedWallet,
} from "@terra-money/wallet-provider";
import { Signer } from "ethers";
import { useSnackbar } from "notistack";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import useTransferSignedVAA from "../hooks/useTransferSignedVAA";
import {
  selectTransferIsRedeeming,
  selectTransferTargetChain,
} from "../store/selectors";
import { setIsRedeeming, setRedeemTx } from "../store/transferSlice";
import {
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  TERRA_TOKEN_BRIDGE_ADDRESS,
} from "../utils/consts";
import parseError from "../utils/parseError";
import { signSendAndConfirm } from "../utils/solana";

async function eth(
  dispatch: any,
  enqueueSnackbar: any,
  signer: Signer,
  signedVAA: Uint8Array,
  isNative: boolean
) {
  dispatch(setIsRedeeming(true));
  try {
    const receipt = isNative
      ? await redeemOnEthNative(ETH_TOKEN_BRIDGE_ADDRESS, signer, signedVAA)
      : await redeemOnEth(ETH_TOKEN_BRIDGE_ADDRESS, signer, signedVAA);
    dispatch(
      setRedeemTx({ id: receipt.transactionHash, block: receipt.blockNumber })
    );
    enqueueSnackbar("Transaction confirmed", { variant: "success" });
  } catch (e) {
    enqueueSnackbar(parseError(e), { variant: "error" });
    dispatch(setIsRedeeming(false));
  }
}

async function solana(
  dispatch: any,
  enqueueSnackbar: any,
  wallet: WalletContextState,
  payerAddress: string, //TODO: we may not need this since we have wallet
  signedVAA: Uint8Array
) {
  dispatch(setIsRedeeming(true));
  try {
    if (!wallet.signTransaction) {
      throw new Error("wallet.signTransaction is undefined");
    }
    const connection = new Connection(SOLANA_HOST, "confirmed");
    await postVaaSolana(
      connection,
      wallet.signTransaction,
      SOL_BRIDGE_ADDRESS,
      payerAddress,
      Buffer.from(signedVAA)
    );
    // TODO: how do we retry in between these steps
    const transaction = await redeemOnSolana(
      connection,
      SOL_BRIDGE_ADDRESS,
      SOL_TOKEN_BRIDGE_ADDRESS,
      payerAddress,
      signedVAA
    );
    const txid = await signSendAndConfirm(wallet, connection, transaction);
    // TODO: didn't want to make an info call we didn't need, can we get the block without it by modifying the above call?
    dispatch(setRedeemTx({ id: txid, block: 1 }));
    enqueueSnackbar("Transaction confirmed", { variant: "success" });
  } catch (e) {
    enqueueSnackbar(parseError(e), { variant: "error" });
    dispatch(setIsRedeeming(false));
  }
}

async function terra(
  dispatch: any,
  enqueueSnackbar: any,
  wallet: ConnectedWallet,
  signedVAA: Uint8Array
) {
  dispatch(setIsRedeeming(true));
  try {
    const msg = await redeemOnTerra(
      TERRA_TOKEN_BRIDGE_ADDRESS,
      wallet.terraAddress,
      signedVAA
    );
    const result = await wallet.post({
      msgs: [msg],
      memo: "Wormhole - Complete Transfer",
    });
    dispatch(
      setRedeemTx({ id: result.result.txhash, block: result.result.height })
    );
    enqueueSnackbar("Transaction confirmed", { variant: "success" });
  } catch (e) {
    enqueueSnackbar(parseError(e), { variant: "error" });
    dispatch(setIsRedeeming(false));
  }
}

export function useHandleRedeem() {
  const dispatch = useDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const targetChain = useSelector(selectTransferTargetChain);
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const { signer } = useEthereumProvider();
  const terraWallet = useConnectedWallet();
  const signedVAA = useTransferSignedVAA();
  const isRedeeming = useSelector(selectTransferIsRedeeming);
  const handleRedeemClick = useCallback(() => {
    if (targetChain === CHAIN_ID_ETH && !!signer && signedVAA) {
      eth(dispatch, enqueueSnackbar, signer, signedVAA, false);
    } else if (
      targetChain === CHAIN_ID_SOLANA &&
      !!solanaWallet &&
      !!solPK &&
      signedVAA
    ) {
      solana(
        dispatch,
        enqueueSnackbar,
        solanaWallet,
        solPK.toString(),
        signedVAA
      );
    } else if (targetChain === CHAIN_ID_TERRA && !!terraWallet && signedVAA) {
      terra(dispatch, enqueueSnackbar, terraWallet, signedVAA);
    } else {
      // enqueueSnackbar("Redeeming on this chain is not yet supported", {
      //   variant: "error",
      // });
    }
  }, [
    dispatch,
    enqueueSnackbar,
    targetChain,
    signer,
    signedVAA,
    solanaWallet,
    solPK,
    terraWallet,
  ]);

  const handleRedeemNativeClick = useCallback(() => {
    if (targetChain === CHAIN_ID_ETH && !!signer && signedVAA) {
      eth(dispatch, enqueueSnackbar, signer, signedVAA, true);
    } else if (
      targetChain === CHAIN_ID_SOLANA &&
      !!solanaWallet &&
      !!solPK &&
      signedVAA
    ) {
      solana(
        dispatch,
        enqueueSnackbar,
        solanaWallet,
        solPK.toString(),
        signedVAA
      );
    } else if (targetChain === CHAIN_ID_TERRA && !!terraWallet && signedVAA) {
      terra(dispatch, enqueueSnackbar, terraWallet, signedVAA); //TODO isNative = true
    } else {
      // enqueueSnackbar("Redeeming on this chain is not yet supported", {
      //   variant: "error",
      // });
    }
  }, [
    dispatch,
    enqueueSnackbar,
    targetChain,
    signer,
    signedVAA,
    solanaWallet,
    solPK,
    terraWallet,
  ]);

  return useMemo(
    () => ({
      handleNativeClick: handleRedeemNativeClick,
      handleClick: handleRedeemClick,
      disabled: !!isRedeeming,
      showLoader: !!isRedeeming,
    }),
    [handleRedeemClick, isRedeeming, handleRedeemNativeClick]
  );
}
