import { createContext } from "react"
import { ChainContracts, endpoints, KnownContracts, NetworkConfig, } from '~/utils/misc/constants';

let defaultNetwork = process.env.GATSBY_DEFAULT_NETWORK || "mainnet"

// ensure the network value is valid
if (!(defaultNetwork in endpoints)) {
    defaultNetwork = defaultNetwork
}
export interface ActiveNetwork {
    name: string
    endpoints: NetworkConfig
    chains: ChainContracts
}
interface NetworkContextI {
    activeNetwork: ActiveNetwork,
    setActiveNetwork: (network: string) => void
}
const NetworkContext = createContext<NetworkContextI>({
    activeNetwork: {
        name: defaultNetwork,
        endpoints: endpoints[defaultNetwork],
        chains: {
            // initalize empty object, will be replaced async by generated data
        }
    },
    setActiveNetwork: (network: string) => { },
})

export { NetworkContext, NetworkContextI }
