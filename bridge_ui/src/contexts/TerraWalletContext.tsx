import { NetworkInfo, WalletProvider } from "@terra-money/wallet-provider";
import { ReactChildren } from "react";
import { CLUSTER } from "../utils/consts";

const mainnet: NetworkInfo = {
  name: "mainnet",
  chainID: "phoenix-1",
  lcd: "https://phoenix-lcd.terra.dev",
  walletconnectID: 1,
};

const classic: NetworkInfo = {
  name: "classic",
  chainID: "columbus-5",
  lcd: "https://columbus-lcd.terra.dev",
  walletconnectID: 2,
}

const testnet: NetworkInfo = {
  name: "testnet",
  chainID: "pisco-1",
  lcd: "https://pisco-lcd.terra.dev",
  walletconnectID: 0,
};

const walletConnectChainIds: Record<number, NetworkInfo> = {
  0: testnet,
  1: mainnet,
  2: classic,
};

export const TerraWalletProvider = ({
  children,
}: {
  children: ReactChildren;
}) => {
  return (
    <WalletProvider
      defaultNetwork={CLUSTER === "testnet" ? testnet : mainnet}
      walletConnectChainIds={walletConnectChainIds}
    >
      {children}
    </WalletProvider>
  );
};
