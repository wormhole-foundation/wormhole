import detectEthereumProvider from "@metamask/detect-provider";
import { ethers } from "ethers";
import React, { ReactChildren, useContext, useEffect, useState } from "react";

type Provider = ethers.providers.Web3Provider | undefined;

const EthereumProviderContext = React.createContext<Provider>(undefined);
export const EthereumProviderProvider = ({
  children,
}: {
  children: ReactChildren;
}) => {
  const [provider, setProvider] = useState<Provider>(undefined);
  useEffect(() => {
    let mounted = true;
    detectEthereumProvider()
      .then((detectedProvider) => {
        if (detectedProvider) {
          if (mounted) {
            const ethersProvider = new ethers.providers.Web3Provider(
              // @ts-ignore
              detectedProvider
            );
            ethersProvider
              .send("eth_requestAccounts", [])
              .then(() => {
                if (mounted) {
                  setProvider(ethersProvider);
                }
              })
              .catch(() => {
                console.error(
                  "An error occurred while requesting eth accounts"
                );
              });
          }
        } else {
          console.log("Please install MetaMask");
        }
      })
      .catch(() => console.log("Please install MetaMask"));
    return () => {
      mounted = false;
    };
  }, []);
  //TODO: useEffect provider.on("network") to refresh on network changes
  //TODO: detect account change
  return (
    <EthereumProviderContext.Provider value={provider}>
      {children}
    </EthereumProviderContext.Provider>
  );
};
export const useEthereumProvider = () => {
  return useContext(EthereumProviderContext);
};
