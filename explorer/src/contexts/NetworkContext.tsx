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
  Network,
  NetworkChains,
  NetworkConfig,
  networks,
} from "../utils/consts";

// Check if window is defined (so if in the browser or in node.js).
const isBrowser = typeof window !== "undefined";

let network: Network;

const envDefaultNetwork = String(process.env.GATSBY_DEFAULT_NETWORK) as Network

// check local storage to get the user's last selected network
let storedNetwork: Network = "" as Network
if (isBrowser) {
  // isBrowser check for Gatsby develop's SSR
  storedNetwork = String(window.localStorage.getItem("networkName")) as Network
}

// validate localstorage data, fallback to .env default, fallback to mainnet.
if (storedNetwork && networks.includes(storedNetwork)) {
  // data from local storage is valid
  network = storedNetwork
} else if (networks.includes(envDefaultNetwork)) {
  // default network from .env is valid
  network = envDefaultNetwork
} else {
  // use mainnet
  network = networks[2];
}

export interface ActiveNetwork {
  name: string;
  endpoints: NetworkConfig;
  chains: ChainContracts;
}

export interface INetworkContext {
  activeNetwork: ActiveNetwork;
  setActiveNetwork: (network: Network) => void;
}

const NetworkContext = createContext<INetworkContext>({
  activeNetwork: {
    name: network,
    endpoints: endpoints[network],
    chains: {
      // initalize empty object, will be replaced async by generated data
    },
  },
  setActiveNetwork: (network: Network) => { },
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
    (network: Network) => {
      async function setNetwork(network: Network) {
        if (isBrowser) {
          // isBrowser check for Gatsby develop's SSR
          window.localStorage.setItem("networkName", network);
        }

        // generate knownContracts if needed
        let contracts = state.knownContracts;
        if (Object.keys(state.knownContracts[network]).length === 0) {
          contracts = await knownContractsPromise;
        }

        setState({
          knownContracts: contracts,
          activeNetwork: {
            name: network,
            endpoints: endpoints[network],
            chains: contracts[network],
          },
        });
      }
      setNetwork(network)
    }, [])

  useMemo(() => setActiveNetwork(network), [])

  return (
    <NetworkContext.Provider value={{ activeNetwork: state.activeNetwork, setActiveNetwork }}>
      {children}
    </NetworkContext.Provider>
  );
};

export const useNetworkContext = () => {
  return useContext(NetworkContext);
};
