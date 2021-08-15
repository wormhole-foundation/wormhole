import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import { Typography } from "@material-ui/core";
import EthereumSignerKey from "./EthereumSignerKey";
import SolanaWalletKey from "./SolanaWalletKey";

function KeyAndBalance({
  chainId,
  balance,
}: {
  chainId: ChainId;
  balance?: string;
}) {
  if (chainId === CHAIN_ID_ETH) {
    return (
      <>
        <EthereumSignerKey />
        <Typography>{balance}</Typography>
      </>
    );
  }
  if (chainId === CHAIN_ID_SOLANA) {
    return (
      <>
        <SolanaWalletKey />
        <Typography>{balance}</Typography>
      </>
    );
  }
  return null;
}

export default KeyAndBalance;
