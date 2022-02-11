import {
  NetworkInfo,
  Wallet,
  WalletProvider,
  useWallet,
  ConnectType,
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
  providerError: string | null;
}

const TerraWalletContext = React.createContext<ITerraWalletContext>({
  connect: () => {},
  disconnect: () => {},
  connected: false,
  wallet: null,
  providerError: null,
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
  const [providerError, setProviderError] = useState<string | null>(null);

  const connect = useCallback(() => {
    if (terraWallet) {
      // TODO: Support other connect types
      if (terraWallet.availableConnectTypes.includes(ConnectType.EXTENSION)) {
        terraWallet.connect(ConnectType.EXTENSION);
        setConnected(true);
        setProviderError(null);
      } else {
        setConnected(false);
        setProviderError("Please install the Terra Station Extension");
      }
      setWallet(terraWallet);
    }
  }, [terraWallet]);

  const disconnect = useCallback(() => {
    setConnected(false);
    setWallet(undefined);
    setProviderError(null);
  }, []);

  const contextValue = useMemo(
    () => ({
      connect,
      disconnect,
      connected,
      wallet: terraWallet,
      providerError,
    }),
    [connect, disconnect, connected, terraWallet, providerError]
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
