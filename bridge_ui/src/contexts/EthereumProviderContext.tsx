import detectEthereumProvider from "@metamask/detect-provider";
import WalletConnectProvider from "@walletconnect/web3-provider";
import { BigNumber, ethers } from "ethers";
import React, {
  ReactChildren,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import metamaskIcon from "../icons/metamask-fox.svg";
import walletconnectIcon from "../icons/walletconnect.svg";

export type Provider = ethers.providers.Web3Provider | undefined;
export type Signer = ethers.Signer | undefined;

export enum ConnectType {
  METAMASK,
  WALLETCONNECT,
}
export interface Connection {
  connectType: ConnectType;
  name: string;
  icon: string;
}

interface IEthereumProviderContext {
  connect(connectType: ConnectType): void;
  disconnect(): void;
  provider: Provider;
  chainId: number | undefined;
  signer: Signer;
  signerAddress: string | undefined;
  providerError: string | null;
  availableConnections: Connection[];
}

const EthereumProviderContext = React.createContext<IEthereumProviderContext>({
  connect: (connectType: ConnectType) => {},
  disconnect: () => {},
  provider: undefined,
  chainId: undefined,
  signer: undefined,
  signerAddress: undefined,
  providerError: null,
  availableConnections: [],
});
export const EthereumProviderProvider = ({
  children,
}: {
  children: ReactChildren;
}) => {
  const [providerError, setProviderError] = useState<string | null>(null);
  const [provider, setProvider] = useState<Provider>(undefined);
  const [chainId, setChainId] = useState<number | undefined>(undefined);
  const [signer, setSigner] = useState<Signer>(undefined);
  const [signerAddress, setSignerAddress] = useState<string | undefined>(
    undefined
  );
  const [availableConnections, setAvailableConnections] = useState<
    Connection[]
  >([]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const connections: Connection[] = [];
      try {
        const detectedProvider = await detectEthereumProvider();
        if (detectedProvider) {
          connections.push({
            connectType: ConnectType.METAMASK,
            name: "MetaMask",
            icon: metamaskIcon,
          });
        }
      } catch (error) {
        console.error(error);
      }
      if (process.env.REACT_APP_INFURA_API_KEY) {
        connections.push({
          connectType: ConnectType.WALLETCONNECT,
          name: "Wallet Connect",
          icon: walletconnectIcon,
        });
      }
      if (!cancelled) {
        setAvailableConnections(connections);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const disconnect = useCallback(() => {
    setProviderError(null);
    setProvider(undefined);
    setChainId(undefined);
    setSigner(undefined);
    setSignerAddress(undefined);
  }, []);

  const connect = useCallback(
    (connectType: ConnectType) => {
      if (connectType === ConnectType.METAMASK) {
        detectEthereumProvider()
          .then((detectedProvider) => {
            if (detectedProvider) {
              const provider = new ethers.providers.Web3Provider(
                // @ts-ignore
                detectedProvider,
                "any"
              );
              provider
                .send("eth_requestAccounts", [])
                .then(() => {
                  setProviderError(null);
                  setProvider(provider);
                  provider
                    .getNetwork()
                    .then((network) => {
                      setChainId(network.chainId);
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
                  // TODO: try using ethers directly
                  // @ts-ignore
                  if (detectedProvider && detectedProvider.on) {
                    // @ts-ignore
                    detectedProvider.on("chainChanged", (chainId) => {
                      try {
                        setChainId(BigNumber.from(chainId).toNumber());
                      } catch (e) {}
                    });
                    // @ts-ignore
                    detectedProvider.on("accountsChanged", (accounts) => {
                      try {
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
                      } catch (e) {}
                    });
                  }
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
      } else if (connectType === ConnectType.WALLETCONNECT) {
        const wcProvider = new WalletConnectProvider({
          infuraId: process.env.REACT_APP_INFURA_API_KEY,
          // Use a custom storageId to support multiple sessions for different chains
          storageId: "walletconnect:ethereum",
        });
        wcProvider
          // Enable session (triggers QR Code modal)
          .enable()
          .then(() => {
            setProviderError(null);
            const provider = new ethers.providers.Web3Provider(
              wcProvider,
              "any"
            );
            provider
              .getNetwork()
              .then((network) => {
                setChainId(network.chainId);
              })
              .catch(() => {
                setProviderError("An error occurred while getting the network");
              });
            wcProvider.on("accountsChanged", (accounts: string[]) => {
              try {
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
              } catch (error) {
                console.error(error);
              }
            });
            wcProvider.on("chainChanged", (chainId: number) => {
              setChainId(chainId);
            });
            wcProvider.on("disconnect", (code: number, reason: string) => {
              disconnect();
            });
            setProvider(provider);
            const signer = provider.getSigner();
            setSigner(signer);
            signer
              .getAddress()
              .then((address) => {
                setSignerAddress(address);
              })
              .catch((error) => {
                setProviderError(
                  "An error occurred while getting the signer address"
                );
                console.error(error);
              });
          })
          .catch((error) => {
            if (error.message !== "User closed modal") {
              setProviderError("Error enabling WalletConnect session");
              console.error(error);
            }
          });
      }
    },
    [disconnect]
  );

  const contextValue = useMemo(
    () => ({
      connect,
      disconnect,
      provider,
      chainId,
      signer,
      signerAddress,
      providerError,
      availableConnections,
    }),
    [
      connect,
      disconnect,
      provider,
      chainId,
      signer,
      signerAddress,
      providerError,
      availableConnections,
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
