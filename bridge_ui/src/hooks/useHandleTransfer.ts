import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  parseSequenceFromLogTerra,
  transferFromEth,
  transferFromSolana,
  transferFromTerra,
} from "@certusone/wormhole-sdk";
import { WalletContextState } from "@solana/wallet-adapter-react";
import { Connection } from "@solana/web3.js";
import {
  ConnectedWallet,
  useConnectedWallet,
} from "@terra-money/wallet-provider";
import { Signer } from "ethers";
import { parseUnits, zeroPad } from "ethers/lib/utils";
import { useSnackbar } from "notistack";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import {
  selectTransferAmount,
  selectTransferIsSendComplete,
  selectTransferIsSending,
  selectTransferIsTargetComplete,
  selectTransferOriginAsset,
  selectTransferOriginChain,
  selectTransferSourceAsset,
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccount,
  selectTransferTargetChain,
} from "../store/selectors";
import { setIsSending, setSignedVAAHex } from "../store/transferSlice";
import { hexToUint8Array, uint8ArrayToHex } from "../utils/array";
import {
  ETH_BRIDGE_ADDRESS,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  TERRA_TOKEN_BRIDGE_ADDRESS,
} from "../utils/consts";
import { getSignedVAAWithRetry } from "../utils/getSignedVAAWithRetry";
import { signSendAndConfirm } from "../utils/solana";
import useTransferTargetAddressHex from "./useTransferTargetAddress";

async function eth(
  dispatch: any,
  enqueueSnackbar: any,
  signer: Signer,
  tokenAddress: string,
  decimals: number,
  amount: string,
  recipientChain: ChainId,
  recipientAddress: Uint8Array
) {
  dispatch(setIsSending(true));
  try {
    const amountParsed = parseUnits(amount, decimals);
    const receipt = await transferFromEth(
      ETH_TOKEN_BRIDGE_ADDRESS,
      signer,
      tokenAddress,
      amountParsed,
      recipientChain,
      recipientAddress
    );
    enqueueSnackbar("Transaction confirmed", { variant: "success" });
    const sequence = parseSequenceFromLogEth(receipt, ETH_BRIDGE_ADDRESS);
    const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
    enqueueSnackbar("Fetching VAA", { variant: "info" });
    const { vaaBytes } = await getSignedVAAWithRetry(
      CHAIN_ID_ETH,
      emitterAddress,
      sequence.toString()
    );
    dispatch(setSignedVAAHex(uint8ArrayToHex(vaaBytes)));
    enqueueSnackbar("Fetched Signed VAA", { variant: "success" });
  } catch (e) {
    console.error(e);
    dispatch(setIsSending(false));
  }
}

async function solana(
  dispatch: any,
  enqueueSnackbar: any,
  wallet: WalletContextState,
  payerAddress: string, //TODO: we may not need this since we have wallet
  fromAddress: string,
  mintAddress: string,
  amount: string,
  decimals: number,
  targetChain: ChainId,
  targetAddress: Uint8Array,
  originAddressStr?: string,
  originChain?: ChainId
) {
  dispatch(setIsSending(true));
  try {
    //TODO: check if token attestation exists on the target chain
    // TODO: share connection in context?
    const connection = new Connection(SOLANA_HOST, "confirmed");
    const amountParsed = parseUnits(amount, decimals).toBigInt();
    const originAddress = originAddressStr
      ? zeroPad(hexToUint8Array(originAddressStr), 32)
      : undefined;
    const transaction = await transferFromSolana(
      connection,
      SOL_BRIDGE_ADDRESS,
      SOL_TOKEN_BRIDGE_ADDRESS,
      payerAddress,
      fromAddress,
      mintAddress,
      amountParsed,
      targetAddress,
      targetChain,
      originAddress,
      originChain
    );
    const txid = await signSendAndConfirm(wallet, connection, transaction);
    enqueueSnackbar("Transaction confirmed", { variant: "success" });
    const info = await connection.getTransaction(txid);
    if (!info) {
      throw new Error("An error occurred while fetching the transaction info");
    }
    enqueueSnackbar("Transaction confirmed", { variant: "success" });
    const sequence = parseSequenceFromLogSolana(info);
    const emitterAddress = await getEmitterAddressSolana(
      SOL_TOKEN_BRIDGE_ADDRESS
    );
    enqueueSnackbar("Fetching VAA", { variant: "info" });
    const { vaaBytes } = await getSignedVAAWithRetry(
      CHAIN_ID_SOLANA,
      emitterAddress,
      sequence
    );

    dispatch(setSignedVAAHex(uint8ArrayToHex(vaaBytes)));
    enqueueSnackbar("Fetched Signed VAA", { variant: "success" });
  } catch (e) {
    console.error(e);
    dispatch(setIsSending(false));
  }
}

async function terra(
  dispatch: any,
  enqueueSnackbar: any,
  wallet: ConnectedWallet,
  asset: string,
  amount: string,
  targetChain: ChainId,
  targetAddress: Uint8Array
) {
  dispatch(setIsSending(true));
  try {
    const msg = await transferFromTerra(
      wallet.terraAddress,
      TERRA_TOKEN_BRIDGE_ADDRESS,
      asset,
      amount,
      targetChain,
      targetAddress
    );
    const result = await wallet.post({
      msgs: [msg],
      memo: "Wormhole - Initiate Transfer",
    });
    enqueueSnackbar("Transaction confirmed", { variant: "success" });
    console.log(result);
    const sequence = parseSequenceFromLogTerra(result);
    console.log(sequence);
    const emitterAddress = await getEmitterAddressTerra(
      TERRA_TOKEN_BRIDGE_ADDRESS
    );
    console.log(emitterAddress);
    enqueueSnackbar("Fetching VAA", { variant: "info" });
    const { vaaBytes } = await getSignedVAAWithRetry(
      CHAIN_ID_TERRA,
      emitterAddress,
      sequence
    );
    enqueueSnackbar("Fetched Signed VAA", { variant: "success" });
    dispatch(setSignedVAAHex(uint8ArrayToHex(vaaBytes)));
  } catch (e) {
    console.error(e);
    dispatch(setIsSending(false));
  }
}

export function useHandleTransfer() {
  const dispatch = useDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const sourceChain = useSelector(selectTransferSourceChain);
  const sourceAsset = useSelector(selectTransferSourceAsset);
  const originChain = useSelector(selectTransferOriginChain);
  const originAsset = useSelector(selectTransferOriginAsset);
  const amount = useSelector(selectTransferAmount);
  const targetChain = useSelector(selectTransferTargetChain);
  const targetAddress = useTransferTargetAddressHex();
  const isTargetComplete = useSelector(selectTransferIsTargetComplete);
  const isSending = useSelector(selectTransferIsSending);
  const isSendComplete = useSelector(selectTransferIsSendComplete);
  const { signer } = useEthereumProvider();
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const terraWallet = useConnectedWallet();
  const sourceParsedTokenAccount = useSelector(
    selectTransferSourceParsedTokenAccount
  );
  const sourceTokenPublicKey = sourceParsedTokenAccount?.publicKey;
  const decimals = sourceParsedTokenAccount?.decimals;
  const disabled = !isTargetComplete || isSending || isSendComplete;
  const handleTransferClick = useCallback(() => {
    // TODO: we should separate state for transaction vs fetching vaa
    // TODO: more generic way of calling these
    if (
      sourceChain === CHAIN_ID_ETH &&
      !!signer &&
      !!sourceAsset &&
      decimals !== undefined &&
      !!targetAddress
    ) {
      eth(
        dispatch,
        enqueueSnackbar,
        signer,
        sourceAsset,
        decimals,
        amount,
        targetChain,
        targetAddress
      );
    } else if (
      sourceChain === CHAIN_ID_SOLANA &&
      !!solanaWallet &&
      !!solPK &&
      !!sourceAsset &&
      !!sourceTokenPublicKey &&
      !!targetAddress &&
      decimals !== undefined
    ) {
      solana(
        dispatch,
        enqueueSnackbar,
        solanaWallet,
        solPK.toString(),
        sourceTokenPublicKey,
        sourceAsset,
        amount,
        decimals,
        targetChain,
        targetAddress,
        originAsset,
        originChain
      );
    } else if (
      sourceChain === CHAIN_ID_TERRA &&
      !!terraWallet &&
      !!sourceAsset &&
      decimals !== undefined &&
      !!targetAddress
    ) {
      terra(
        dispatch,
        enqueueSnackbar,
        terraWallet,
        sourceAsset,
        amount,
        targetChain,
        targetAddress
      );
    } else {
      // enqueueSnackbar("Transfers from this chain are not yet supported", {
      //   variant: "error",
      // });
    }
  }, [
    dispatch,
    enqueueSnackbar,
    sourceChain,
    signer,
    solanaWallet,
    solPK,
    terraWallet,
    sourceTokenPublicKey,
    sourceAsset,
    amount,
    decimals,
    targetChain,
    targetAddress,
    originAsset,
    originChain,
  ]);
  return useMemo(
    () => ({
      handleClick: handleTransferClick,
      disabled,
      showLoader: isSending,
    }),
    [handleTransferClick, disabled, isSending]
  );
}
