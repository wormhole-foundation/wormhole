
const addresses = {
    solana: {
        token: ['Solana Token Bridge', process.env.SOL_TOKEN_BRIDGE],
        bridge: ['Solana Core Bridge', process.env.SOL_CORE_BRIDGE],
    },
    ethereum: {
        token: ['Ethereum Token Bridge', process.env.ETH_TOKEN_BRIDGE],
        core: ['Ethereum Core Bridge', process.env.ETH_CORE_BRIDGE],
    },
    terra: {
        token: ['Terra Token Bridge', process.env.LUN_TOKEN_BRIDGE],
        core: ['Terra Core Bridge', process.env.LUN_CORE_BRIDGE],
    },
    bsc: {
        token: ['BSC Token Bridge', process.env.BSC_TOKEN_BRIDGE],
        core: ['BSC Core Bridge', process.env.BSC_CORE_BRIDGE],
    },
}
enum ChainID {
    Solana,
    Ethereum,
    Terra,
    'Binance Smart Chain'
}


export { addresses, ChainID }
