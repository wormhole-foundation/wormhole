import {
  NetworkInfo,
  Wallet,
  WalletProvider,
  useWallet,
} from "@terra-money/wallet-provider";
import React, {
  ReactChildren,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";
import { TERRA_HOST } from "../utils/consts";

const mainnet = {
  name: "mainnet",
  chainID: "columbus-4",
  lcd: "https://lcd.terra.dev",
};

const localnet = {
  name: "localnet",
  chainID: "localnet",
  lcd: TERRA_HOST.URL,
};

const walletConnectChainIds: Record<number, NetworkInfo> = {
  0: localnet,
  1: mainnet,
};

interface ITerraWalletContext {
  connect(): void;
  disconnect(): void;
  connected: boolean;
  wallet: any;
}

const TerraWalletContext = React.createContext<ITerraWalletContext>({
  connect: () => {},
  disconnect: () => {},
  connected: false,
  wallet: null,
});

export const TerraWalletWrapper = ({
  children,
}: {
  children: ReactChildren;
}) => {
  // TODO: Use wallet instead of useConnectedWallet.
  const terraWallet = useWallet();
  const [, setWallet] = useState<Wallet | undefined>(undefined);
  const [connected, setConnected] = useState(false);

  const connect = useCallback(() => {
    const CHROME_EXTENSION = 1;
    if (terraWallet) {
      terraWallet.connect(terraWallet.availableConnectTypes[CHROME_EXTENSION]);
      setWallet(terraWallet);
      setConnected(true);
    }
  }, [terraWallet]);

  const disconnect = useCallback(() => {
    setConnected(false);
    setWallet(undefined);
  }, []);

  const contextValue = useMemo(
    () => ({
      connect,
      disconnect,
      connected,
      wallet: terraWallet,
    }),
    [connect, disconnect, connected, terraWallet]
  );

  return (
    <TerraWalletContext.Provider value={contextValue}>
      {children}
    </TerraWalletContext.Provider>
  );
};

export const TerraWalletProvider = ({
  children,
}: {
  children: ReactChildren;
}) => {
  return (
    <WalletProvider
      defaultNetwork={localnet}
      walletConnectChainIds={walletConnectChainIds}
    >
      <TerraWalletWrapper>{children}</TerraWalletWrapper>
    </WalletProvider>
  );
};

export const useTerraWallet = () => {
  return useContext(TerraWalletContext);
};
