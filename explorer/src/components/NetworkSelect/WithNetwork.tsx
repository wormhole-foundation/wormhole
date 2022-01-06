import React from 'react';
import { NetworkContext } from '~/components/NetworkSelect'
import { NetworkContextI } from './network-context'
import { endpoints, KnownContracts, knownContractsPromise, NetworkChains, networks } from '~/utils/misc/constants';

// Check if window is defined (so if in the browser or in node.js).
const isBrowser = typeof window !== "undefined"

const defaultNetwork = process.env.GATSBY_DEFAULT_NETWORK || "mainnet"

interface NetworkContextState extends NetworkContextI {
    knownContracts: NetworkChains
}
const WithNetwork = (WrappedComponent: React.FC<any>) => {

    return class extends React.Component<{}, NetworkContextState> {
        constructor(props: any) {
            super(props)

            let network: string | undefined | null = ""
            if (isBrowser) {
                // isBrowser check for Gatsby develop's SSR
                network = window.localStorage.getItem("networkName")
            }
            if (!network || !networks.includes(network)) {
                network = defaultNetwork
            }

            this.state = {
                // knownContracts are generated async and added to state
                knownContracts: {
                    "devnet": {},
                    "testnet": {},
                    "mainnet": {}
                },
                activeNetwork: {
                    name: network,
                    endpoints: endpoints[network],
                    chains: {
                        // chains are generated async and added to state
                    }
                },
                setActiveNetwork: this.setActiveNetwork,
            };
            this.setActiveNetwork(network)
        }

        setActiveNetwork = async (network: string) => {
            if (isBrowser) {
                // isBrowser check for Gatsby develop's SSR
                window.localStorage.setItem("networkName", network)
            }

            // generate knownContracts if needed
            let contracts = this.state.knownContracts
            if (!this.state.knownContracts.devent) {
                contracts = await knownContractsPromise
                this.setState(() => ({
                    knownContracts: contracts
                }))
            }

            this.setState(() => ({
                activeNetwork: {
                    name: network,
                    endpoints: endpoints[network],
                    chains: contracts[network],
                }
            }));
        }
        render() {
            return (
                <NetworkContext.Provider value={this.state}>
                    <WrappedComponent {...this.props} />
                </NetworkContext.Provider>
            )
        }
    }
}

export default WithNetwork
