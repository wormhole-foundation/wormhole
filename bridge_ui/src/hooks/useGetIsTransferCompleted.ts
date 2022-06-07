import {
  CHAIN_ID_ALGORAND,
  CHAIN_ID_SOLANA,
  getIsTransferCompletedAlgorand,
  getIsTransferCompletedEth,
  getIsTransferCompletedSolana,
  getIsTransferCompletedTerra,
  isEVMChain,
  isTerraChain,
} from "@certusone/wormhole-sdk";
import { Connection } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import algosdk from "algosdk";
import { useEffect, useState } from "react";
import { useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import {
  selectTransferIsRecovery,
  selectTransferTargetAddressHex,
  selectTransferTargetChain,
} from "../store/selectors";
import {
  ALGORAND_HOST,
  ALGORAND_TOKEN_BRIDGE_ID,
  getEvmChainId,
  getTokenBridgeAddressForChain,
  SOLANA_HOST,
  getTerraGasPricesUrl,
  getTerraConfig,
} from "../utils/consts";
import useIsWalletReady from "./useIsWalletReady";
import useTransferSignedVAA from "./useTransferSignedVAA";

/**
 * @param recoveryOnly Only fire when in recovery mode
 */
export default function useGetIsTransferCompleted(
  recoveryOnly: boolean,
  pollFrequency?: number
): {
  isTransferCompletedLoading: boolean;
  isTransferCompleted: boolean;
} {
  const [isLoading, setIsLoading] = useState(false);
  const [isTransferCompleted, setIsTransferCompleted] = useState(false);

  const isRecovery = useSelector(selectTransferIsRecovery);
  const targetAddress = useSelector(selectTransferTargetAddressHex);
  const targetChain = useSelector(selectTransferTargetChain);

  const { isReady } = useIsWalletReady(targetChain, false);
  const { provider, chainId: evmChainId } = useEthereumProvider();
  const signedVAA = useTransferSignedVAA();

  const hasCorrectEvmNetwork = evmChainId === getEvmChainId(targetChain);
  const shouldFire = !recoveryOnly || isRecovery;
  const [pollState, setPollState] = useState(pollFrequency);

  console.log(
    "Executing get transfer completed",
    isTransferCompleted,
    pollState
  );

  useEffect(() => {
    let cancelled = false;
    if (pollFrequency && !isLoading && !isTransferCompleted) {
      setTimeout(() => {
        if (!cancelled) {
          setPollState((prevState) => (prevState || 0) + 1);
        }
      }, pollFrequency);
    }
    return () => {
      cancelled = true;
    };
  }, [pollFrequency, isLoading, isTransferCompleted]);

  useEffect(() => {
    if (!shouldFire) {
      return;
    }

    let cancelled = false;
    let transferCompleted = false;
    if (targetChain && targetAddress && signedVAA && isReady) {
      if (isEVMChain(targetChain) && hasCorrectEvmNetwork && provider) {
        setIsLoading(true);
        (async () => {
          try {
            transferCompleted = await getIsTransferCompletedEth(
              getTokenBridgeAddressForChain(targetChain),
              provider,
              signedVAA
            );
          } catch (error) {
            console.error(error);
          }
          if (!cancelled) {
            setIsTransferCompleted(transferCompleted);
            setIsLoading(false);
          }
        })();
      } else if (targetChain === CHAIN_ID_SOLANA) {
        setIsLoading(true);
        (async () => {
          try {
            const connection = new Connection(SOLANA_HOST, "confirmed");
            transferCompleted = await getIsTransferCompletedSolana(
              getTokenBridgeAddressForChain(targetChain),
              signedVAA,
              connection
            );
          } catch (error) {
            console.error(error);
          }
          if (!cancelled) {
            setIsTransferCompleted(transferCompleted);
            setIsLoading(false);
          }
        })();
      } else if (isTerraChain(targetChain)) {
        setIsLoading(true);
        (async () => {
          try {
            const lcdClient = new LCDClient(getTerraConfig(targetChain));
            transferCompleted = await getIsTransferCompletedTerra(
              getTokenBridgeAddressForChain(targetChain),
              signedVAA,
              lcdClient,
              getTerraGasPricesUrl(targetChain)
            );
          } catch (error) {
            console.error(error);
          }
          if (!cancelled) {
            setIsTransferCompleted(transferCompleted);
            setIsLoading(false);
          }
        })();
      } else if (targetChain === CHAIN_ID_ALGORAND) {
        setIsLoading(true);
        (async () => {
          try {
            const algodClient = new algosdk.Algodv2(
              ALGORAND_HOST.algodToken,
              ALGORAND_HOST.algodServer,
              ALGORAND_HOST.algodPort
            );
            transferCompleted = await getIsTransferCompletedAlgorand(
              algodClient,
              ALGORAND_TOKEN_BRIDGE_ID,
              signedVAA
            );
          } catch (error) {
            console.error(error);
          }
          if (!cancelled) {
            setIsTransferCompleted(transferCompleted);
            setIsLoading(false);
          }
        })();
      }
    }
    return () => {
      cancelled = true;
    };
  }, [
    shouldFire,
    hasCorrectEvmNetwork,
    targetChain,
    targetAddress,
    signedVAA,
    isReady,
    provider,
    pollState,
  ]);

  return { isTransferCompletedLoading: isLoading, isTransferCompleted };
}
