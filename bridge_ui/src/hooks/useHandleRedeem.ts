import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  postVaaSolana,
  redeemOnEth,
  redeemOnSolana,
} from "@certusone/wormhole-sdk";
import { WalletContextState } from "@solana/wallet-adapter-react";
import { Connection } from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import {
  ConnectedWallet,
  useConnectedWallet,
} from "@terra-money/wallet-provider";
import { Signer } from "ethers";
import { fromUint8Array } from "js-base64";
import { useSnackbar } from "notistack";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import useTransferSignedVAA from "../hooks/useTransferSignedVAA";
import {
  selectTransferIsRedeeming,
  selectTransferIsSourceAssetWormholeWrapped,
  selectTransferOriginChain,
  selectTransferTargetAsset,
  selectTransferTargetChain,
} from "../store/selectors";
import { reset, setIsRedeeming } from "../store/transferSlice";
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
  signedVAA: Uint8Array
) {
  dispatch(setIsRedeeming(true));
  try {
    await redeemOnEth(ETH_TOKEN_BRIDGE_ADDRESS, signer, signedVAA);
    dispatch(reset());
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
  signedVAA: Uint8Array,
  isSolanaNative: boolean,
  mintAddress?: string // TODO: read the signedVAA and create the account if it doesn't exist
) {
  dispatch(setIsRedeeming(true));
  try {
    // TODO: share connection in context?
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
      signedVAA,
      isSolanaNative,
      mintAddress
    );
    await signSendAndConfirm(wallet, connection, transaction);
    dispatch(reset());
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
    await wallet.post({
      msgs: [
        new MsgExecuteContract(
          wallet.terraAddress,
          TERRA_TOKEN_BRIDGE_ADDRESS,
          {
            submit_vaa: {
              data: fromUint8Array(signedVAA),
            },
          },
          { uluna: 1000 }
        ),
      ],
      memo: "Complete Transfer",
    });
    dispatch(reset());
    enqueueSnackbar("Transaction confirmed", { variant: "success" });
  } catch (e) {
    enqueueSnackbar(parseError(e), { variant: "error" });
    dispatch(setIsRedeeming(false));
  }
}

export function useHandleRedeem() {
  const dispatch = useDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const isSourceAssetWormholeWrapped = useSelector(
    selectTransferIsSourceAssetWormholeWrapped
  );
  const originChain = useSelector(selectTransferOriginChain);
  const targetChain = useSelector(selectTransferTargetChain);
  const targetAsset = useSelector(selectTransferTargetAsset);
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const { signer } = useEthereumProvider();
  const terraWallet = useConnectedWallet();
  const signedVAA = useTransferSignedVAA();
  const isRedeeming = useSelector(selectTransferIsRedeeming);
  const handleRedeemClick = useCallback(() => {
    if (targetChain === CHAIN_ID_ETH && !!signer && signedVAA) {
      eth(dispatch, enqueueSnackbar, signer, signedVAA);
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
        signedVAA,
        !!isSourceAssetWormholeWrapped && originChain === CHAIN_ID_SOLANA,
        targetAsset || undefined
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
    isSourceAssetWormholeWrapped,
    originChain,
    targetAsset,
  ]);
  return useMemo(
    () => ({
      handleClick: handleRedeemClick,
      disabled: !!isRedeeming,
      showLoader: !!isRedeeming,
    }),
    [handleRedeemClick, isRedeeming]
  );
}
