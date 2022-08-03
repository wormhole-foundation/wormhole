import {
  attestFromAlgorand,
  attestFromEth,
  attestFromSolana,
  attestFromTerra,
  ChainId,
  CHAIN_ID_ALGORAND,
  CHAIN_ID_KLAYTN,
  CHAIN_ID_SOLANA,
  getEmitterAddressAlgorand,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
  getSignedVAAWithRetry,
  isEVMChain,
  isTerraChain,
  parseSequenceFromLogAlgorand,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  parseSequenceFromLogTerra,
  TerraChainId,
  uint8ArrayToHex,
} from "@certusone/wormhole-sdk";
import { Alert } from "@material-ui/lab";
import { WalletContextState } from "@solana/wallet-adapter-react";
import { Connection, PublicKey } from "@solana/web3.js";
import {
  ConnectedWallet,
  useConnectedWallet,
} from "@terra-money/wallet-provider";
import algosdk from "algosdk";
import { Signer } from "ethers";
import { useSnackbar } from "notistack";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useAlgorandContext } from "../contexts/AlgorandWalletContext";
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
import { signSendAndConfirmAlgorand } from "../utils/algorand";
import {
  ALGORAND_BRIDGE_ID,
  ALGORAND_HOST,
  ALGORAND_TOKEN_BRIDGE_ID,
  getBridgeAddressForChain,
  getTokenBridgeAddressForChain,
  SOLANA_HOST,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  WORMHOLE_RPC_HOSTS,
} from "../utils/consts";
import parseError from "../utils/parseError";
import { signSendAndConfirm } from "../utils/solana";
import { postWithFees, waitForTerraExecution } from "../utils/terra";

async function algo(
  dispatch: any,
  enqueueSnackbar: any,
  senderAddr: string,
  sourceAsset: string
) {
  dispatch(setIsSending(true));
  try {
    const algodClient = new algosdk.Algodv2(
      ALGORAND_HOST.algodToken,
      ALGORAND_HOST.algodServer,
      ALGORAND_HOST.algodPort
    );
    const txs = await attestFromAlgorand(
      algodClient,
      ALGORAND_TOKEN_BRIDGE_ID,
      ALGORAND_BRIDGE_ID,
      senderAddr,
      BigInt(sourceAsset)
    );
    const result = await signSendAndConfirmAlgorand(algodClient, txs);
    const sequence = parseSequenceFromLogAlgorand(result);
    // TODO: fill these out correctly
    dispatch(
      setAttestTx({
        id: txs[txs.length - 1].tx.txID(),
        block: result["confirmed-round"],
      })
    );
    enqueueSnackbar(null, {
      content: <Alert severity="success">Transaction confirmed</Alert>,
    });
    const emitterAddress = getEmitterAddressAlgorand(ALGORAND_TOKEN_BRIDGE_ID);
    enqueueSnackbar(null, {
      content: <Alert severity="info">Fetching VAA</Alert>,
    });
    const { vaaBytes } = await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      CHAIN_ID_ALGORAND,
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

async function evm(
  dispatch: any,
  enqueueSnackbar: any,
  signer: Signer,
  sourceAsset: string,
  chainId: ChainId
) {
  dispatch(setIsSending(true));
  try {
    // Klaytn requires specifying gasPrice
    const overrides =
      chainId === CHAIN_ID_KLAYTN
        ? { gasPrice: (await signer.getGasPrice()).toString() }
        : {};
    const receipt = await attestFromEth(
      getTokenBridgeAddressForChain(chainId),
      signer,
      sourceAsset,
      overrides
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
      WORMHOLE_RPC_HOSTS,
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
      WORMHOLE_RPC_HOSTS,
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
  feeDenom: string,
  chainId: TerraChainId
) {
  dispatch(setIsSending(true));
  try {
    const tokenBridgeAddress = getTokenBridgeAddressForChain(chainId);
    const msg = await attestFromTerra(
      tokenBridgeAddress,
      wallet.terraAddress,
      asset
    );
    const result = await postWithFees(
      wallet,
      [msg],
      "Create Wrapped",
      [feeDenom],
      chainId
    );
    const info = await waitForTerraExecution(result, chainId);
    dispatch(setAttestTx({ id: info.txhash, block: info.height }));
    enqueueSnackbar(null, {
      content: <Alert severity="success">Transaction confirmed</Alert>,
    });
    const sequence = parseSequenceFromLogTerra(info);
    if (!sequence) {
      throw new Error("Sequence not found");
    }
    const emitterAddress = await getEmitterAddressTerra(tokenBridgeAddress);
    enqueueSnackbar(null, {
      content: <Alert severity="info">Fetching VAA</Alert>,
    });
    const { vaaBytes } = await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
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
  const { accounts: algoAccounts } = useAlgorandContext();
  const disabled = !isTargetComplete || isSending || isSendComplete;
  const handleAttestClick = useCallback(() => {
    if (isEVMChain(sourceChain) && !!signer) {
      evm(dispatch, enqueueSnackbar, signer, sourceAsset, sourceChain);
    } else if (sourceChain === CHAIN_ID_SOLANA && !!solanaWallet && !!solPK) {
      solana(dispatch, enqueueSnackbar, solPK, sourceAsset, solanaWallet);
    } else if (isTerraChain(sourceChain) && !!terraWallet) {
      terra(
        dispatch,
        enqueueSnackbar,
        terraWallet,
        sourceAsset,
        terraFeeDenom,
        sourceChain
      );
    } else if (sourceChain === CHAIN_ID_ALGORAND && algoAccounts[0]) {
      algo(dispatch, enqueueSnackbar, algoAccounts[0].address, sourceAsset);
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
    algoAccounts,
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
