import {
  ChainId,
  CHAIN_ID_ACALA,
  CHAIN_ID_KARURA,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  createWrappedOnEth,
  createWrappedOnSolana,
  createWrappedOnTerra,
  isEVMChain,
  postVaaSolanaWithRetry,
  updateWrappedOnEth,
  updateWrappedOnSolana,
  updateWrappedOnTerra,
} from "@certusone/wormhole-sdk";
import { Alert } from "@material-ui/lab";
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
import { setCreateTx, setIsCreating } from "../store/attestSlice";
import {
  selectAttestIsCreating,
  selectAttestTargetChain,
  selectTerraFeeDenom,
} from "../store/selectors";
import {
  ACALA_HOST,
  getTokenBridgeAddressForChain,
  KARURA_HOST,
  MAX_VAA_UPLOAD_RETRIES_SOLANA,
  SOLANA_HOST,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  TERRA_TOKEN_BRIDGE_ADDRESS,
} from "../utils/consts";
import { getKaruraGasParams } from "../utils/karura";
import parseError from "../utils/parseError";
import { signSendAndConfirm } from "../utils/solana";
import { postWithFees } from "../utils/terra";
import useAttestSignedVAA from "./useAttestSignedVAA";

async function evm(
  dispatch: any,
  enqueueSnackbar: any,
  signer: Signer,
  signedVAA: Uint8Array,
  chainId: ChainId,
  shouldUpdate: boolean
) {
  dispatch(setIsCreating(true));
  try {
    // Karura and Acala need gas params for contract deploys
    const overrides =
      chainId === CHAIN_ID_KARURA
        ? await getKaruraGasParams(KARURA_HOST)
        : chainId === CHAIN_ID_ACALA
        ? await getKaruraGasParams(ACALA_HOST)
        : {};
    const receipt = shouldUpdate
      ? await updateWrappedOnEth(
          getTokenBridgeAddressForChain(chainId),
          signer,
          signedVAA,
          overrides
        )
      : await createWrappedOnEth(
          getTokenBridgeAddressForChain(chainId),
          signer,
          signedVAA,
          overrides
        );
    dispatch(
      setCreateTx({ id: receipt.transactionHash, block: receipt.blockNumber })
    );
    enqueueSnackbar(null, {
      content: <Alert severity="success">Transaction confirmed</Alert>,
    });
  } catch (e) {
    enqueueSnackbar(null, {
      content: <Alert severity="error">{parseError(e)}</Alert>,
    });
    dispatch(setIsCreating(false));
  }
}

async function solana(
  dispatch: any,
  enqueueSnackbar: any,
  wallet: WalletContextState,
  payerAddress: string, // TODO: we may not need this since we have wallet
  signedVAA: Uint8Array,
  shouldUpdate: boolean
) {
  dispatch(setIsCreating(true));
  try {
    if (!wallet.signTransaction) {
      throw new Error("wallet.signTransaction is undefined");
    }
    const connection = new Connection(SOLANA_HOST, "confirmed");
    await postVaaSolanaWithRetry(
      connection,
      wallet.signTransaction,
      SOL_BRIDGE_ADDRESS,
      payerAddress,
      Buffer.from(signedVAA),
      MAX_VAA_UPLOAD_RETRIES_SOLANA
    );
    const transaction = shouldUpdate
      ? await updateWrappedOnSolana(
          connection,
          SOL_BRIDGE_ADDRESS,
          SOL_TOKEN_BRIDGE_ADDRESS,
          payerAddress,
          signedVAA
        )
      : await createWrappedOnSolana(
          connection,
          SOL_BRIDGE_ADDRESS,
          SOL_TOKEN_BRIDGE_ADDRESS,
          payerAddress,
          signedVAA
        );
    const txid = await signSendAndConfirm(wallet, connection, transaction);
    // TODO: didn't want to make an info call we didn't need, can we get the block without it by modifying the above call?
    dispatch(setCreateTx({ id: txid, block: 1 }));
    enqueueSnackbar(null, {
      content: <Alert severity="success">Transaction confirmed</Alert>,
    });
  } catch (e) {
    enqueueSnackbar(null, {
      content: <Alert severity="error">{parseError(e)}</Alert>,
    });
    dispatch(setIsCreating(false));
  }
}

async function terra(
  dispatch: any,
  enqueueSnackbar: any,
  wallet: ConnectedWallet,
  signedVAA: Uint8Array,
  shouldUpdate: boolean,
  feeDenom: string
) {
  dispatch(setIsCreating(true));
  try {
    const msg = shouldUpdate
      ? await updateWrappedOnTerra(
          TERRA_TOKEN_BRIDGE_ADDRESS,
          wallet.terraAddress,
          signedVAA
        )
      : await createWrappedOnTerra(
          TERRA_TOKEN_BRIDGE_ADDRESS,
          wallet.terraAddress,
          signedVAA
        );
    const result = await postWithFees(
      wallet,
      [msg],
      "Wormhole - Create Wrapped",
      [feeDenom]
    );
    dispatch(
      setCreateTx({ id: result.result.txhash, block: result.result.height })
    );
    enqueueSnackbar(null, {
      content: <Alert severity="success">Transaction confirmed</Alert>,
    });
  } catch (e) {
    enqueueSnackbar(null, {
      content: <Alert severity="error">{parseError(e)}</Alert>,
    });
    dispatch(setIsCreating(false));
  }
}

export function useHandleCreateWrapped(shouldUpdate: boolean) {
  const dispatch = useDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const targetChain = useSelector(selectAttestTargetChain);
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const signedVAA = useAttestSignedVAA();
  const isCreating = useSelector(selectAttestIsCreating);
  const { signer } = useEthereumProvider();
  const terraWallet = useConnectedWallet();
  const terraFeeDenom = useSelector(selectTerraFeeDenom);
  const handleCreateClick = useCallback(() => {
    if (isEVMChain(targetChain) && !!signer && !!signedVAA) {
      evm(
        dispatch,
        enqueueSnackbar,
        signer,
        signedVAA,
        targetChain,
        shouldUpdate
      );
    } else if (
      targetChain === CHAIN_ID_SOLANA &&
      !!solanaWallet &&
      !!solPK &&
      !!signedVAA
    ) {
      solana(
        dispatch,
        enqueueSnackbar,
        solanaWallet,
        solPK.toString(),
        signedVAA,
        shouldUpdate
      );
    } else if (targetChain === CHAIN_ID_TERRA && !!terraWallet && !!signedVAA) {
      terra(
        dispatch,
        enqueueSnackbar,
        terraWallet,
        signedVAA,
        shouldUpdate,
        terraFeeDenom
      );
    } else {
      // enqueueSnackbar(
      //   "Creating wrapped tokens on this chain is not yet supported",
      //   {
      //     variant: "error",
      //   }
      // );
    }
  }, [
    dispatch,
    enqueueSnackbar,
    targetChain,
    solanaWallet,
    solPK,
    terraWallet,
    signedVAA,
    signer,
    shouldUpdate,
    terraFeeDenom,
  ]);
  return useMemo(
    () => ({
      handleClick: handleCreateClick,
      disabled: !!isCreating,
      showLoader: !!isCreating,
    }),
    [handleCreateClick, isCreating]
  );
}
