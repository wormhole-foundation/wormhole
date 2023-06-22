export const DEVELOPMENT_KMD_TOKEN = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
export const DEVELOPMENT_KMD_HOST  = "localhost"
export const DEVELOPMENT_KMD_PORT  = 4002

export type AlgorandServerConnectionConfig = {
    token: string,
    server: string,
    port: number | string
}

export type ExecutionEnvironmentConfig = {
    algod: AlgorandServerConnectionConfig,
    kmd?: AlgorandServerConnectionConfig
}

export type AccountSigningData = {
    mnemonic: string
} | {
    secretKey: Uint8Array
}

export type Mnemonic = string
export type TestExecutionEnvironmentConfig = ExecutionEnvironmentConfig & {
    masterAccount: Mnemonic
}

export const BETANET_CONFIG: TestExecutionEnvironmentConfig = {
    algod: {token: "", server: "https://node.betanet.algoexplorerapi.io", port: ""},
    masterAccount: "rate firm prefer portion innocent public large original fit shoulder solve scorpion battle end jealous off pause inner toddler year grab chaos result about capital"
}
/**
 * Path: Direct path to sandbox executable, in this example
 * since I use Windows OS i had to add sh command in front of full path
 * In my case I downloaded sandbox in disk D:\
 */
export const LOCAL_CONFIG: TestExecutionEnvironmentConfig = {
    algod: {
        token: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        server: "http://localhost",
        port: 4001
    },
    // Public Key: HL6A24OGJX4FDZT36HOQ6VWJDF6GW3IEWB4FXB4OH5FQKVI46HZBZOZFAM
    masterAccount: "general foster traffic label come once baby attract travel nose clap mystery want problem beyond side wing bridge drastic one sun diet trigger absent fossil"
}

export type WormholeConfig = {
    coreBridgeAppId: bigint,
    tokenBridgeAppId: bigint
}

export const WORMHOLE_CONFIG_MAINNET: WormholeConfig = {
    coreBridgeAppId: BigInt("0"),
    tokenBridgeAppId: BigInt("0")
}
export const WORMHOLE_CONFIG_TESTNET: WormholeConfig = {
    coreBridgeAppId: BigInt("86525623"),
    tokenBridgeAppId: BigInt("86525641")
}
export const WORMHOLE_CONFIG_DEVNET: WormholeConfig = {
    coreBridgeAppId: BigInt("1004"),
    tokenBridgeAppId: BigInt("1006")
}
