import {
  ChainId,
  CHAIN_ID_ALGORAND,
  CHAIN_ID_SOLANA,
  isEVMChain,
  isTerraChain,
} from "@certusone/wormhole-sdk";
import AlgorandWalletKey from "./AlgorandWalletKey";
import EthereumSignerKey from "./EthereumSignerKey";
import SolanaWalletKey from "./SolanaWalletKey";
import TerraWalletKey from "./TerraWalletKey";

function KeyAndBalance({ chainId }: { chainId: ChainId }) {
  if (isEVMChain(chainId)) {
    return <EthereumSignerKey chainId={chainId} />;
  }
  if (chainId === CHAIN_ID_SOLANA) {
    return <SolanaWalletKey />;
  }
  if (isTerraChain(chainId)) {
    return <TerraWalletKey />;
  }
  if (chainId === CHAIN_ID_ALGORAND) {
    return <AlgorandWalletKey />;
  }
  return null;
}

export default KeyAndBalance;
