import { getEmitterAddressEth, getEmitterAddressSolana, getEmitterAddressTerra } from "@certusone/wormhole-sdk";

export const chainEnums = ['', 'Solana', 'Ethereum', 'Terra', 'BSC', 'Polygon']
// type chainNames = "solana" | "ethereum" | "terra" | "bsc"
export interface ChainIDs {
    "solana": 1,
    "ethereum": 2,
    "terra": 3,
    "bsc": 4,
    "polygon": 5
}
export const chainIDs: ChainIDs = {
    "solana": 1,
    "ethereum": 2,
    "terra": 3,
    "bsc": 4,
    "polygon": 5
}

export enum ChainID {
    "unknown",
    Solana,
    Ethereum,
    Terra,
    'Binance Smart Chain',
    Polygon,
}

export const METADATA_REPLACE = new RegExp("\u0000", "g");

// Gatsby only includes environment variables that are explictly referenced, it does the substitution at build time.
// Created this map as a work around to access them dynamically (ie. process.env[someKeyName]).
const envVarMap: { [name: string]: string | undefined } = {
    GATSBY_DEVNET_SOLANA_CORE_BRIDGE: process.env.GATSBY_DEVNET_SOLANA_CORE_BRIDGE,
    GATSBY_DEVNET_SOLANA_TOKEN_BRIDGE: process.env.GATSBY_DEVNET_SOLANA_TOKEN_BRIDGE,
    GATSBY_DEVNET_SOLANA_NFT_BRIDGE: process.env.GATSBY_DEVNET_SOLANA_NFT_BRIDGE,
    GATSBY_DEVNET_ETHEREUM_CORE_BRIDGE: process.env.GATSBY_DEVNET_ETHEREUM_CORE_BRIDGE,
    GATSBY_DEVNET_ETHEREUM_TOKEN_BRIDGE: process.env.GATSBY_DEVNET_ETHEREUM_TOKEN_BRIDGE,
    GATSBY_DEVNET_ETHEREUM_NFT_BRIDGE: process.env.GATSBY_DEVNET_ETHEREUM_NFT_BRIDGE,
    GATSBY_DEVNET_TERRA_CORE_BRIDGE: process.env.GATSBY_DEVNET_TERRA_CORE_BRIDGE,
    GATSBY_DEVNET_TERRA_TOKEN_BRIDGE: process.env.GATSBY_DEVNET_TERRA_TOKEN_BRIDGE,
    GATSBY_DEVNET_TERRA_NFT_BRIDGE: process.env.GATSBY_DEVNET_TERRA_NFT_BRIDGE,
    GATSBY_DEVNET_BSC_CORE_BRIDGE: process.env.GATSBY_DEVNET_BSC_CORE_BRIDGE,
    GATSBY_DEVNET_BSC_TOKEN_BRIDGE: process.env.GATSBY_DEVNET_BSC_TOKEN_BRIDGE,
    GATSBY_DEVNET_BSC_NFT_BRIDGE: process.env.GATSBY_DEVNET_BSC_NFT_BRIDGE,
    GATSBY_DEVNET_POLYGON_CORE_BRIDGE: process.env.GATSBY_DEVNET_POLYGON_CORE_BRIDGE,
    GATSBY_DEVNET_POLYGON_TOKEN_BRIDGE: process.env.GATSBY_DEVNET_POLYGON_TOKEN_BRIDGE,
    GATSBY_DEVNET_POLYGON_NFT_BRIDGE: process.env.GATSBY_DEVNET_POLYGON_NFT_BRIDGE,

    GATSBY_TESTNET_SOLANA_CORE_BRIDGE: process.env.GATSBY_TESTNET_SOLANA_CORE_BRIDGE,
    GATSBY_TESTNET_SOLANA_TOKEN_BRIDGE: process.env.GATSBY_TESTNET_SOLANA_TOKEN_BRIDGE,
    GATSBY_TESTNET_SOLANA_NFT_BRIDGE: process.env.GATSBY_TESTNET_SOLANA_NFT_BRIDGE,
    GATSBY_TESTNET_ETHEREUM_CORE_BRIDGE: process.env.GATSBY_TESTNET_ETHEREUM_CORE_BRIDGE,
    GATSBY_TESTNET_ETHEREUM_TOKEN_BRIDGE: process.env.GATSBY_TESTNET_ETHEREUM_TOKEN_BRIDGE,
    GATSBY_TESTNET_ETHEREUM_NFT_BRIDGE: process.env.GATSBY_TESTNET_ETHEREUM_NFT_BRIDGE,
    GATSBY_TESTNET_TERRA_CORE_BRIDGE: process.env.GATSBY_TESTNET_TERRA_CORE_BRIDGE,
    GATSBY_TESTNET_TERRA_TOKEN_BRIDGE: process.env.GATSBY_TESTNET_TERRA_TOKEN_BRIDGE,
    GATSBY_TESTNET_TERRA_NFT_BRIDGE: process.env.GATSBY_TESTNET_TERRA_NFT_BRIDGE,
    GATSBY_TESTNET_BSC_CORE_BRIDGE: process.env.GATSBY_TESTNET_BSC_CORE_BRIDGE,
    GATSBY_TESTNET_BSC_TOKEN_BRIDGE: process.env.GATSBY_TESTNET_BSC_TOKEN_BRIDGE,
    GATSBY_TESTNET_BSC_NFT_BRIDGE: process.env.GATSBY_TESTNET_BSC_NFT_BRIDGE,
    GATSBY_TESTNET_POLYGON_CORE_BRIDGE: process.env.GATSBY_TESTNET_POLYGON_CORE_BRIDGE,
    GATSBY_TESTNET_POLYGON_TOKEN_BRIDGE: process.env.GATSBY_TESTNET_POLYGON_TOKEN_BRIDGE,
    GATSBY_TESTNET_POLYGON_NFT_BRIDGE: process.env.GATSBY_TESTNET_POLYGON_NFT_BRIDGE,

    GATSBY_MAINNET_SOLANA_CORE_BRIDGE: process.env.GATSBY_MAINNET_SOLANA_CORE_BRIDGE,
    GATSBY_MAINNET_SOLANA_TOKEN_BRIDGE: process.env.GATSBY_MAINNET_SOLANA_TOKEN_BRIDGE,
    GATSBY_MAINNET_SOLANA_NFT_BRIDGE: process.env.GATSBY_MAINNET_SOLANA_NFT_BRIDGE,
    GATSBY_MAINNET_ETHEREUM_CORE_BRIDGE: process.env.GATSBY_MAINNET_ETHEREUM_CORE_BRIDGE,
    GATSBY_MAINNET_ETHEREUM_TOKEN_BRIDGE: process.env.GATSBY_MAINNET_ETHEREUM_TOKEN_BRIDGE,
    GATSBY_MAINNET_ETHEREUM_NFT_BRIDGE: process.env.GATSBY_MAINNET_ETHEREUM_NFT_BRIDGE,
    GATSBY_MAINNET_TERRA_CORE_BRIDGE: process.env.GATSBY_MAINNET_TERRA_CORE_BRIDGE,
    GATSBY_MAINNET_TERRA_TOKEN_BRIDGE: process.env.GATSBY_MAINNET_TERRA_TOKEN_BRIDGE,
    GATSBY_MAINNET_TERRA_NFT_BRIDGE: process.env.GATSBY_MAINNET_TERRA_NFT_BRIDGE,
    GATSBY_MAINNET_BSC_CORE_BRIDGE: process.env.GATSBY_MAINNET_BSC_CORE_BRIDGE,
    GATSBY_MAINNET_BSC_TOKEN_BRIDGE: process.env.GATSBY_MAINNET_BSC_TOKEN_BRIDGE,
    GATSBY_MAINNET_BSC_NFT_BRIDGE: process.env.GATSBY_MAINNET_BSC_NFT_BRIDGE,
    GATSBY_MAINNET_POLYGON_CORE_BRIDGE: process.env.GATSBY_MAINNET_POLYGON_CORE_BRIDGE,
    GATSBY_MAINNET_POLYGON_TOKEN_BRIDGE: process.env.GATSBY_MAINNET_POLYGON_TOKEN_BRIDGE,
    GATSBY_MAINNET_POLYGON_NFT_BRIDGE: process.env.GATSBY_MAINNET_POLYGON_NFT_BRIDGE,
}

export interface KnownContracts {
    "Token Bridge": string
    "Core Bridge": string
    "NFT Bridge": string
    [address: string]: string
}
export interface ChainContracts {
    [chainName: string]: KnownContracts
}
export interface NetworkChains {
    [networkName: string]: ChainContracts
}

const getEmitterAddressEVM = (address: string) => Promise.resolve(getEmitterAddressEth(address))
const getEmitterAddress: { [chainName: string]: (address: string) => Promise<string> } = {
    "solana": getEmitterAddressSolana,
    "ethereum": getEmitterAddressEVM,
    "terra": getEmitterAddressTerra,
    "bsc": getEmitterAddressEVM,
    "polygon": getEmitterAddressEVM,
}

// the keys used for creating the map of contract addresses of each chain, on each network.
export const networks = ["devnet", "testnet", "mainnet"]
const contractTypes = ["Core", "Token", "NFT"]
const chainNames = Object.keys(chainIDs)

export const knownContractsPromise = networks.reduce<Promise<NetworkChains>>(async (promisedAccum, network) => {
    // Create a data structure to access contract addresses by network, then chain,
    // so that for the network picker.
    // Index by address and name, so you can easily get at the data either way.
    // {
    //     devnet: {
    //         solana: {
    //             'Token Bridge': String(process.env.DEVNET_SOLANA_TOKEN_BRIDGE),
    //             String(process.env.DEVNET_SOLANA_TOKEN_BRIDGE): 'Token Bridge'
    //         },
    //         ethereum: {
    //             'Token Bridge': String(process.env.DEVNET_ETHEREUM_TOKEN_BRIDGE),
    //              String(process.env.DEVNET_ETHEREUM_TOKEN_BRIDGE): 'Token Bridge'
    //         },
    //         terra: {
    //             'Token Bridge': String(process.env.DEVNET_TERRA_TOKEN_BRIDGE),
    //              String(process.env.DEVNET_TERRA_TOKEN_BRIDGE): 'Token Bridge'
    //         },
    //         bsc: {
    //             'Token Bridge': String(process.env.DEVNET_BSC_TOKEN_BRIDGE),
    //              String(process.env.DEVNET_BSC_TOKEN_BRIDGE): 'Token Bridge'
    //         },
    //     },
    //     testnet: {...},
    //     mainnet: {...}
    // }
    const accum = await promisedAccum
    accum[network] = await chainNames.reduce<Promise<ChainContracts>>(async (promisedSubAccum, chainName) => {
        const subAccum = await promisedSubAccum
        subAccum[chainName] = await contractTypes.reduce<Promise<KnownContracts>>(async (promisedContractsOfChain, contractType) => {
            const contractsOfChain = await promisedContractsOfChain
            const envVarName = ['GATSBY', network.toUpperCase(), chainName.toUpperCase(), contractType.toUpperCase(), 'BRIDGE'].join("_")
            let address = envVarMap[envVarName]
            if (!address) throw `missing environment variable: ${envVarName}`
            const desc = `${contractType} Bridge`
            // index by: description, contract address, and emitter address
            try {
                const emitterAddress = await getEmitterAddress[chainName](address)
                contractsOfChain[emitterAddress] = desc
            } catch (_) {
                console.log('failed getting emitterAddress for: ', address)
            }
            if (chainName != "solana") {
                address = address.toLowerCase()
            }
            contractsOfChain[desc] = address
            contractsOfChain[address] = desc
            return contractsOfChain
        }, Promise.resolve(Object()))
        return subAccum
    }, Promise.resolve(Object()))
    return accum
}, Promise.resolve(Object()))


export interface NetworkConfig {
    bigtableFunctionsBase: string
    guardianRpcBase: string
}
export const endpoints: { [network: string]: NetworkConfig } = {
    devnet: {
        bigtableFunctionsBase: String(process.env.GATSBY_BIGTABLE_FUNCTIONS_DEVNET_BASE_URL),
        guardianRpcBase: String(process.env.GATSBY_GUARDIAN_DEVNET_RPC_URL)
    },
    testnet: {
        bigtableFunctionsBase: String(process.env.GATSBY_BIGTABLE_FUNCTIONS_TESTNET_BASE_URL),
        guardianRpcBase: String(process.env.GATSBY_GUARDIAN_TESTNET_RPC_URL)
    },
    mainnet: {
        bigtableFunctionsBase: String(process.env.GATSBY_BIGTABLE_FUNCTIONS_MAINNET_BASE_URL),
        guardianRpcBase: String(process.env.GATSBY_GUARDIAN_MAINNET_RPC_URL)
    }
}

export const BRIDGE_URL = "https://wormholebridge.com"
export const DOCS_URL = "https://docs.wormholenetwork.com"
export const DISCORD_URL = "https://discord.gg/xsT8qrHAvV"
export const GITHUB_URL = "https://github.com/certusone/wormhole"
export const JOBS_URL = "https://boards.greenhouse.io/wormhole"
export const MEDIUM_URL = "http://wormholecrypto.medium.com"
export const TELEGRAM_URL = "https://t.me/wormholecrypto"
export const TWITTER_URL = "https://twitter.com/wormholecrypto"
