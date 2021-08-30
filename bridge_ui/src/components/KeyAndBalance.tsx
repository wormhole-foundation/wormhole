import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { Typography } from "@material-ui/core";
import EthereumSignerKey from "./EthereumSignerKey";
import SolanaWalletKey from "./SolanaWalletKey";
import TerraWalletKey from "./TerraWalletKey";

function KeyAndBalance({
  chainId,
  balance,
}: {
  chainId: ChainId;
  balance?: string;
}) {
  const balanceString = balance ? "Balance: " + balance : balance;
  if (chainId === CHAIN_ID_ETH) {
    return (
      <>
        <EthereumSignerKey />
        <Typography>{balanceString}</Typography>
      </>
    );
  }
  if (chainId === CHAIN_ID_SOLANA) {
    return (
      <>
        <SolanaWalletKey />
        <Typography>{balanceString}</Typography>
      </>
    );
  }
  if (chainId === CHAIN_ID_TERRA) {
    return (
      <>
        <TerraWalletKey />
        <Typography>{balanceString}</Typography>
      </>
    );
  }
  return null;
}

export default KeyAndBalance;
