import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { useConnectedWallet } from "@terra-money/wallet-provider";
import { useMemo } from "react";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import { CLUSTER, ETH_NETWORK_CHAIN_ID } from "../utils/consts";

const createWalletStatus = (isReady: boolean, statusMessage: string = "") => ({
  isReady,
  statusMessage,
});

function useIsWalletReady(chainId: ChainId): {
  isReady: boolean;
  statusMessage: string;
} {
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const terraWallet = useConnectedWallet();
  const hasTerraWallet = !!terraWallet;
  const {
    provider,
    signerAddress,
    chainId: ethChainId,
  } = useEthereumProvider();
  const hasEthInfo = !!provider && !!signerAddress;
  const hasCorrectEthNetwork = ethChainId === ETH_NETWORK_CHAIN_ID;

  return useMemo(() => {
    if (chainId === CHAIN_ID_TERRA && hasTerraWallet) {
      // TODO: terraWallet does not update on wallet changes
      return createWalletStatus(true);
    }
    if (chainId === CHAIN_ID_SOLANA && solPK) {
      return createWalletStatus(true);
    }
    if (chainId === CHAIN_ID_ETH && hasEthInfo) {
      if (hasCorrectEthNetwork) {
        return createWalletStatus(true);
      } else {
        return createWalletStatus(
          false,
          `Wallet is not connected to ${CLUSTER}. Expected Chain ID: ${ETH_NETWORK_CHAIN_ID}`
        );
      }
    }
    //TODO bsc
    return createWalletStatus(false, "Wallet not connected");
  }, [chainId, hasTerraWallet, solPK, hasEthInfo, hasCorrectEthNetwork]);
}

export default useIsWalletReady;
