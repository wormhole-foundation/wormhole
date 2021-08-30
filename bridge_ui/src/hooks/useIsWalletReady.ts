import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { useConnectedWallet } from "@terra-money/wallet-provider";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";

function useIsWalletReady(chainId: ChainId) {
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const terraWallet = useConnectedWallet();
  const { provider, signerAddress } = useEthereumProvider();

  let output = false;
  if (chainId === CHAIN_ID_TERRA && terraWallet) {
    output = true;
  } else if (chainId === CHAIN_ID_SOLANA && solPK) {
    output = true;
  } else if (chainId === CHAIN_ID_ETH && provider && signerAddress) {
    output = true;
  }
  //TODO bsc

  return output;
}

export default useIsWalletReady;
