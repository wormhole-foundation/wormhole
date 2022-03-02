import { NetworkInfo, WalletProvider } from "@terra-money/wallet-provider";
import { ReactChildren } from "react";
import { CLUSTER } from "../utils/consts";

const mainnet: NetworkInfo = {
  name: "mainnet",
  chainID: "columbus-5",
  lcd: "https://lcd.terra.dev",
};

const testnet: NetworkInfo = {
  name: "testnet",
  chainID: "bombay-12",
  lcd: "https://bombay-lcd.terra.dev",
};

const walletConnectChainIds: Record<number, NetworkInfo> = {
  0: testnet,
  1: mainnet,
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
