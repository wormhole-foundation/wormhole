import React, {
  createContext,
  ReactChildren,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";
import {
  ChainContracts,
  endpoints,
  knownContractsPromise,
  NetworkChains,
  NetworkConfig,
  networks,
} from "../utils/consts";

// Check if window is defined (so if in the browser or in node.js).
const isBrowser = typeof window !== "undefined";

let defaultNetwork = process.env.GATSBY_DEFAULT_NETWORK || "mainnet";
let network = "";
if (isBrowser) {
  // isBrowser check for Gatsby develop's SSR
  network = window.localStorage.getItem("networkName") || "";
}
if (!network || !networks.includes(network)) {
  network = defaultNetwork;
}

// ensure the network value is valid
if (!(defaultNetwork in endpoints)) {
  defaultNetwork = defaultNetwork;
}

export interface ActiveNetwork {
  name: string;
  endpoints: NetworkConfig;
  chains: ChainContracts;
}

export interface INetworkContext {
  knownContracts: NetworkChains;
  activeNetwork: ActiveNetwork;
  setActiveNetwork: (network: keyof NetworkChains) => void;
}

const NetworkContext = createContext<INetworkContext>({
  knownContracts: {
    devnet: {},
    testnet: {},
    mainnet: {},
  },
  activeNetwork: {
    name: defaultNetwork,
    endpoints: endpoints[defaultNetwork],
    chains: {
      // initalize empty object, will be replaced async by generated data
    },
  },
  setActiveNetwork: (network: keyof NetworkChains) => {},
});

export const NetworkContextProvider = ({
  children,
}: {
  children: ReactChildren;
}) => {
  const [state, setState] = useState({
    // knownContracts are generated async and added to state
    knownContracts: {
      devnet: {},
      testnet: {},
      mainnet: {},
    } as NetworkChains,
    activeNetwork: {
      name: network,
      endpoints: endpoints[network],
      chains: {
        // chains are generated async and added to state
      },
    } as ActiveNetwork,
  });
  const setActiveNetwork = useCallback(
    (network: keyof NetworkChains) => {
      let cancelled = false;
      (async () => {
        if (isBrowser) {
          // isBrowser check for Gatsby develop's SSR
          window.localStorage.setItem("networkName", network);
        }

        // generate knownContracts if needed
        let contracts = state.knownContracts;
        if (Object.keys(state.knownContracts[network]).length === 0) {
          contracts = await knownContractsPromise;
        }
        if (cancelled) return;
        setState({
          knownContracts: contracts,
          activeNetwork: {
            name: network,
            endpoints: endpoints[network],
            chains: contracts[network],
          },
        });
      })();
      return () => {
        cancelled = true;
      };
    },
    [state]
  );
  const value = useMemo(
    () => ({
      ...state,
      setActiveNetwork,
    }),
    [state]
  );
  return (
    <NetworkContext.Provider value={value}>{children}</NetworkContext.Provider>
  );
};

export const useNetworkContext = () => {
  return useContext(NetworkContext);
};
