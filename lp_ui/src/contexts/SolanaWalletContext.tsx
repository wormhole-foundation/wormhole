import { WalletDialogProvider } from "@solana/wallet-adapter-material-ui";
import { useWallet, WalletProvider } from "@solana/wallet-adapter-react";
import { getPhantomWallet } from "@solana/wallet-adapter-wallets";
import React, { FC, useMemo } from "react";

export const SolanaWalletProvider: FC = (props) => {
  // @solana/wallet-adapter-wallets includes all the adapters but supports tree shaking --
  // Only the wallets you want to instantiate here will be compiled into your application
  const wallets = useMemo(() => {
    return [getPhantomWallet()];
  }, []);

  return (
    <WalletProvider wallets={wallets}>
      <WalletDialogProvider>{props.children}</WalletDialogProvider>
    </WalletProvider>
  );
};

export const useSolanaWallet = useWallet;
