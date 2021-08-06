import detectEthereumProvider from "@metamask/detect-provider";
import { ethers } from "ethers";
import React, {
  ReactChildren,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";

type Provider = ethers.providers.Web3Provider | undefined;
type Signer = ethers.Signer | undefined;
type Network = ethers.providers.Network | undefined;

interface IEthereumProviderContext {
  connect(): void;
  disconnect(): void;
  provider: Provider;
  network: Network;
  signer: Signer;
  signerAddress: string | undefined;
  providerError: string | null;
}

const EthereumProviderContext = React.createContext<IEthereumProviderContext>({
  connect: () => {},
  disconnect: () => {},
  provider: undefined,
  network: undefined,
  signer: undefined,
  signerAddress: undefined,
  providerError: null,
});
export const EthereumProviderProvider = ({
  children,
}: {
  children: ReactChildren;
}) => {
  const [providerError, setProviderError] = useState<string | null>(null);
  const [provider, setProvider] = useState<Provider>(undefined);
  const [network, setNetwork] = useState<Network>(undefined);
  const [signer, setSigner] = useState<Signer>(undefined);
  const [signerAddress, setSignerAddress] = useState<string | undefined>(
    undefined
  );
  const connect = useCallback(() => {
    setProviderError(null);
    detectEthereumProvider()
      .then((detectedProvider) => {
        if (detectedProvider) {
          const provider = new ethers.providers.Web3Provider(
            // @ts-ignore
            detectedProvider,
            "any" //TODO: should we only allow homestead? env perhaps?
          );
          provider
            .send("eth_requestAccounts", [])
            .then(() => {
              setProviderError(null);
              setProvider(provider);
              provider
                .getNetwork()
                .then((network) => {
                  setNetwork(network);
                })
                .catch(() => {
                  setProviderError(
                    "An error occurred while getting the network"
                  );
                });
              const signer = provider.getSigner();
              setSigner(signer);
              signer
                .getAddress()
                .then((address) => {
                  setSignerAddress(address);
                })
                .catch(() => {
                  setProviderError(
                    "An error occurred while getting the signer address"
                  );
                });
            })
            .catch(() => {
              setProviderError(
                "An error occurred while requesting eth accounts"
              );
            });
        } else {
          setProviderError("Please install MetaMask");
        }
      })
      .catch(() => {
        setProviderError("Please install MetaMask");
      });
  }, []);
  const disconnect = useCallback(() => {
    setProviderError(null);
    setProvider(undefined);
    setNetwork(undefined);
    setSigner(undefined);
    setSignerAddress(undefined);
  }, []);
  //TODO: detect account change
  const contextValue = useMemo(
    () => ({
      connect,
      disconnect,
      provider,
      network,
      signer,
      signerAddress,
      providerError,
    }),
    [
      connect,
      disconnect,
      provider,
      network,
      signer,
      signerAddress,
      providerError,
    ]
  );
  return (
    <EthereumProviderContext.Provider value={contextValue}>
      {children}
    </EthereumProviderContext.Provider>
  );
};
export const useEthereumProvider = () => {
  return useContext(EthereumProviderContext);
};
