import {
  attestFromEth,
  attestFromSolana,
  attestFromTerra,
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
  isEVMChain,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  parseSequenceFromLogTerra,
  uint8ArrayToHex,
} from "@certusone/wormhole-sdk";
import { Alert } from "@material-ui/lab";
import { WalletContextState } from "@solana/wallet-adapter-react";
import { Connection, PublicKey } from "@solana/web3.js";
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
import {
  setAttestTx,
  setIsSending,
  setSignedVAAHex,
} from "../store/attestSlice";
import {
  selectAttestIsSendComplete,
  selectAttestIsSending,
  selectAttestIsTargetComplete,
  selectAttestSourceAsset,
  selectAttestSourceChain,
  selectTerraFeeDenom,
} from "../store/selectors";
import {
  getBridgeAddressForChain,
  getTokenBridgeAddressForChain,
  SOLANA_HOST,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  TERRA_TOKEN_BRIDGE_ADDRESS,
} from "../utils/consts";
import { getSignedVAAWithRetry } from "../utils/getSignedVAAWithRetry";
import parseError from "../utils/parseError";
import { signSendAndConfirm } from "../utils/solana";
import { postWithFees, waitForTerraExecution } from "../utils/terra";

async function evm(
  dispatch: any,
  enqueueSnackbar: any,
  signer: Signer,
  sourceAsset: string,
  chainId: ChainId
) {
  dispatch(setIsSending(true));
  try {
    const receipt = await attestFromEth(
      getTokenBridgeAddressForChain(chainId),
      signer,
      sourceAsset
    );
    dispatch(
      setAttestTx({ id: receipt.transactionHash, block: receipt.blockNumber })
    );
    enqueueSnackbar(null, {
      content: <Alert severity="success">Transaction confirmed</Alert>,
    });
    const sequence = parseSequenceFromLogEth(
      receipt,
      getBridgeAddressForChain(chainId)
    );
    const emitterAddress = getEmitterAddressEth(
      getTokenBridgeAddressForChain(chainId)
    );
    enqueueSnackbar(null, {
      content: <Alert severity="info">Fetching VAA</Alert>,
    });
    const { vaaBytes } = await getSignedVAAWithRetry(
      chainId,
      emitterAddress,
      sequence
    );
    dispatch(setSignedVAAHex(uint8ArrayToHex(vaaBytes)));
    enqueueSnackbar(null, {
      content: <Alert severity="success">Fetched Signed VAA</Alert>,
    });
  } catch (e) {
    console.error(e);
    enqueueSnackbar(null, {
      content: <Alert severity="error">{parseError(e)}</Alert>,
    });
    dispatch(setIsSending(false));
  }
}

async function solana(
  dispatch: any,
  enqueueSnackbar: any,
  solPK: PublicKey,
  sourceAsset: string,
  wallet: WalletContextState
) {
  dispatch(setIsSending(true));
  try {
    const connection = new Connection(SOLANA_HOST, "confirmed");
    const transaction = await attestFromSolana(
      connection,
      SOL_BRIDGE_ADDRESS,
      SOL_TOKEN_BRIDGE_ADDRESS,
      solPK.toString(),
      sourceAsset
    );
    const txid = await signSendAndConfirm(wallet, connection, transaction);
    enqueueSnackbar(null, {
      content: <Alert severity="success">Transaction confirmed</Alert>,
    });
    const info = await connection.getTransaction(txid);
    if (!info) {
      // TODO: error state
      throw new Error("An error occurred while fetching the transaction info");
    }
    dispatch(setAttestTx({ id: txid, block: info.slot }));
    const sequence = parseSequenceFromLogSolana(info);
    const emitterAddress = await getEmitterAddressSolana(
      SOL_TOKEN_BRIDGE_ADDRESS
    );
    enqueueSnackbar(null, {
      content: <Alert severity="info">Fetching VAA</Alert>,
    });
    const { vaaBytes } = await getSignedVAAWithRetry(
      CHAIN_ID_SOLANA,
      emitterAddress,
      sequence
    );
    dispatch(setSignedVAAHex(uint8ArrayToHex(vaaBytes)));
    enqueueSnackbar(null, {
      content: <Alert severity="success">Fetched Signed VAA</Alert>,
    });
  } catch (e) {
    console.error(e);
    enqueueSnackbar(null, {
      content: <Alert severity="error">{parseError(e)}</Alert>,
    });
    dispatch(setIsSending(false));
  }
}

async function terra(
  dispatch: any,
  enqueueSnackbar: any,
  wallet: ConnectedWallet,
  asset: string,
  feeDenom: string
) {
  dispatch(setIsSending(true));
  try {
    const msg = await attestFromTerra(
      TERRA_TOKEN_BRIDGE_ADDRESS,
      wallet.terraAddress,
      asset
    );
    const result = await postWithFees(wallet, [msg], "Create Wrapped", [
      feeDenom,
    ]);
    const info = await waitForTerraExecution(result);
    dispatch(setAttestTx({ id: info.txhash, block: info.height }));
    enqueueSnackbar(null, {
      content: <Alert severity="success">Transaction confirmed</Alert>,
    });
    const sequence = parseSequenceFromLogTerra(info);
    if (!sequence) {
      throw new Error("Sequence not found");
    }
    const emitterAddress = await getEmitterAddressTerra(
      TERRA_TOKEN_BRIDGE_ADDRESS
    );
    enqueueSnackbar(null, {
      content: <Alert severity="info">Fetching VAA</Alert>,
    });
    const { vaaBytes } = await getSignedVAAWithRetry(
      CHAIN_ID_TERRA,
      emitterAddress,
      sequence
    );
    dispatch(setSignedVAAHex(uint8ArrayToHex(vaaBytes)));
    enqueueSnackbar(null, {
      content: <Alert severity="success">Fetched Signed VAA</Alert>,
    });
  } catch (e) {
    console.error(e);
    enqueueSnackbar(null, {
      content: <Alert severity="error">{parseError(e)}</Alert>,
    });
    dispatch(setIsSending(false));
  }
}

export function useHandleAttest() {
  const dispatch = useDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const sourceChain = useSelector(selectAttestSourceChain);
  const sourceAsset = useSelector(selectAttestSourceAsset);
  const isTargetComplete = useSelector(selectAttestIsTargetComplete);
  const isSending = useSelector(selectAttestIsSending);
  const isSendComplete = useSelector(selectAttestIsSendComplete);
  const { signer } = useEthereumProvider();
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const terraWallet = useConnectedWallet();
  const terraFeeDenom = useSelector(selectTerraFeeDenom);
  const disabled = !isTargetComplete || isSending || isSendComplete;
  const handleAttestClick = useCallback(() => {
    if (isEVMChain(sourceChain) && !!signer) {
      evm(dispatch, enqueueSnackbar, signer, sourceAsset, sourceChain);
    } else if (sourceChain === CHAIN_ID_SOLANA && !!solanaWallet && !!solPK) {
      solana(dispatch, enqueueSnackbar, solPK, sourceAsset, solanaWallet);
    } else if (sourceChain === CHAIN_ID_TERRA && !!terraWallet) {
      terra(dispatch, enqueueSnackbar, terraWallet, sourceAsset, terraFeeDenom);
    } else {
    }
  }, [
    dispatch,
    enqueueSnackbar,
    sourceChain,
    signer,
    solanaWallet,
    solPK,
    terraWallet,
    sourceAsset,
    terraFeeDenom,
  ]);
  return useMemo(
    () => ({
      handleClick: handleAttestClick,
      disabled,
      showLoader: isSending,
    }),
    [handleAttestClick, disabled, isSending]
  );
}
