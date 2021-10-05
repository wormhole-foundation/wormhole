import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { hexlify, hexStripZeros } from "@ethersproject/bytes";
import { useConnectedWallet } from "@terra-money/wallet-provider";
import { useMemo } from "react";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import { CLUSTER, getEvmChainId } from "../utils/consts";
import { isEVMChain } from "../utils/ethereum";

const createWalletStatus = (
  isReady: boolean,
  statusMessage: string = "",
  walletAddress?: string
) => ({
  isReady,
  statusMessage,
  walletAddress,
});

function useIsWalletReady(chainId: ChainId): {
  isReady: boolean;
  statusMessage: string;
  walletAddress?: string;
} {
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const terraWallet = useConnectedWallet();
  const hasTerraWallet = !!terraWallet;
  const {
    provider,
    signerAddress,
    chainId: evmChainId,
  } = useEthereumProvider();
  const hasEthInfo = !!provider && !!signerAddress;
  const correctEvmNetwork = getEvmChainId(chainId);
  const hasCorrectEvmNetwork = evmChainId === correctEvmNetwork;

  return useMemo(() => {
    if (
      chainId === CHAIN_ID_TERRA &&
      hasTerraWallet &&
      terraWallet?.walletAddress
    ) {
      // TODO: terraWallet does not update on wallet changes
      return createWalletStatus(true, undefined, terraWallet.walletAddress);
    }
    if (chainId === CHAIN_ID_SOLANA && solPK) {
      return createWalletStatus(true, undefined, solPK.toString());
    }
    if (isEVMChain(chainId) && hasEthInfo && signerAddress) {
      if (hasCorrectEvmNetwork) {
        return createWalletStatus(true, undefined, signerAddress);
      } else {
        if (provider && correctEvmNetwork) {
          try {
            provider.send("wallet_switchEthereumChain", [
              { chainId: hexStripZeros(hexlify(correctEvmNetwork)) },
            ]);
          } catch (e) {}
        }
        return createWalletStatus(
          false,
          `Wallet is not connected to ${CLUSTER}. Expected Chain ID: ${correctEvmNetwork}`,
          undefined
        );
      }
    }
    //TODO bsc
    return createWalletStatus(false, "Wallet not connected");
  }, [
    chainId,
    hasTerraWallet,
    solPK,
    hasEthInfo,
    correctEvmNetwork,
    hasCorrectEvmNetwork,
    provider,
    signerAddress,
    terraWallet,
  ]);
}

export default useIsWalletReady;
