import { Typography } from "@material-ui/core";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import useEthereumBalance from "../hooks/useEthereumBalance";
import useSolanaBalance from "../hooks/useSolanaBalance";
import { ChainId, CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "../utils/consts";
import EthereumSignerKey from "./EthereumSignerKey";
import SolanaWalletKey from "./SolanaWalletKey";

function KeyAndBalance({
  chainId,
  tokenAddress,
}: {
  chainId: ChainId;
  tokenAddress?: string;
}) {
  // TODO: more generic way to get balance
  const provider = useEthereumProvider();
  const ethBalance = useEthereumBalance(
    tokenAddress,
    provider,
    chainId === CHAIN_ID_ETH
  );
  const { wallet: solWallet } = useSolanaWallet();
  const solPK = solWallet?.publicKey;
  const { uiAmountString: solBalance } = useSolanaBalance(
    tokenAddress,
    solPK,
    chainId === CHAIN_ID_SOLANA
  );
  if (chainId === CHAIN_ID_ETH) {
    return (
      <>
        <EthereumSignerKey />
        <Typography>{ethBalance}</Typography>
      </>
    );
  }
  if (chainId === CHAIN_ID_SOLANA) {
    return (
      <>
        <SolanaWalletKey />
        <Typography>{solBalance}</Typography>
      </>
    );
  }
  return null;
}

export default KeyAndBalance;
