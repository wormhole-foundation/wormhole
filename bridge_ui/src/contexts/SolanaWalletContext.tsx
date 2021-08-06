import Wallet from "@project-serum/sol-wallet-adapter";
import React, {
  ReactChildren,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";
import { SOLANA_HOST } from "../utils/consts";

interface ISolanaWalletContext {
  connect(): void;
  disconnect(): void;
  connected: boolean;
  wallet: Wallet | undefined;
}

const SolanaWalletContext = React.createContext<ISolanaWalletContext>({
  connect: () => {},
  disconnect: () => {},
  connected: false,
  wallet: undefined,
});
export const SolanaWalletProvider = ({
  children,
}: {
  children: ReactChildren;
}) => {
  const [wallet, setWallet] = useState<Wallet | undefined>(undefined);
  const [connected, setConnected] = useState(false);
  const connect = useCallback(() => {
    const wallet = new Wallet("https://www.sollet.io", SOLANA_HOST);
    setWallet(wallet);
    wallet.on("connect", () => {
      setConnected(true);
    });
    wallet.on("disconnect", () => {
      console.log("disconnected");
      setConnected(false);
      setWallet(undefined);
    });
    wallet.connect();
  }, []);
  const disconnect = useCallback(() => {
    wallet?.disconnect();
  }, [wallet]);
  const contextValue = useMemo(
    () => ({ connect, disconnect, connected, wallet }),
    [connect, disconnect, wallet, connected]
  );
  return (
    <SolanaWalletContext.Provider value={contextValue}>
      {children}
    </SolanaWalletContext.Provider>
  );
};
export const useSolanaWallet = () => {
  return useContext(SolanaWalletContext);
};
