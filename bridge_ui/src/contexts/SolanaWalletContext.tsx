import React, {
  ReactChildren,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import Wallet from "@project-serum/sol-wallet-adapter";
import { SOLANA_HOST } from "../utils/consts";

interface ISolanaWalletContext {
  connected: boolean;
  wallet: Wallet | undefined;
}

const getDefaultWallet = () => new Wallet("https://www.sollet.io", SOLANA_HOST);
const SolanaWalletContext = React.createContext<ISolanaWalletContext>({
  connected: false,
  wallet: undefined,
});
export const SolanaWalletProvider = ({
  children,
}: {
  children: ReactChildren;
}) => {
  const wallet = useMemo(getDefaultWallet, []);
  const [connected, setConnected] = useState(false);
  useEffect(() => {
    wallet.on("connect", () => {
      setConnected(true);
      console.log("Connected to wallet " + wallet.publicKey?.toBase58());
    });
    wallet.on("disconnect", () => {
      setConnected(false);
      console.log("Disconnected from wallet");
    });
    wallet.connect();
    return () => {
      wallet.disconnect();
    };
  }, [wallet]);
  console.log(`Connected state: ${connected}`);
  //TODO: useEffect to refresh on network changes
  // ensure users of the context refresh on connect state change
  const contextValue = useMemo(
    () => ({ connected, wallet }),
    [wallet, connected]
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
