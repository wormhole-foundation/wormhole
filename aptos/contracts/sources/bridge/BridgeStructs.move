module Wormhole::BridgeStructs {

    struct Transfer<phantom CoinType> has key, store{
        // PayloadID uint8 = 1
        payloadID: u8,
        // Amount being transferred (big-endian uint256)
        amount: u128,
        // Address of the token. Left-zero-padded if shorter than 32 bytes
        tokenAddress: vector<u8>,
        // Chain ID of the token
        tokenChain: u64,//should be u16
        // Address of the recipient. Left-zero-padded if shorter than 32 bytes
        to: vector<u8>,
        // Chain ID of the recipient
        toChain: u64,
        // Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
        fee: u128, //should be u256
    }

    struct TransferWithPayload<phantom CoinType> has key, store {
        // PayloadID uint8 = 3
        payloadID: u8,
        // Amount being transferred (big-endian uint256)
        amount: u128, //should be u256
        // Address of the token. Left-zero-padded if shorter than 32 bytes
        tokenAddress: vector<u8>,
        // Chain ID of the token
        tokenChain: u64,
        // Address of the recipient. Left-zero-padded if shorter than 32 bytes
        to: vector<u8>,
        // Chain ID of the recipient
        toChain: u64, //should be u16
        // Address of the message sender. Left-zero-padded if shorter than 32 bytes
        fromAddress: vector<u8>,
        // An arbitrary payload
        payload: vector<u8>,
    }

    struct TransferResult<phantom CoinType> has key, store {
        // Chain ID of the token
        tokenChain: u64, // should be u16
        // Address of the token. Left-zero-padded if shorter than 32 bytes
        tokenAddress: vector<u8>, 
        // Amount being transferred (big-endian uint256)
        normalizedAmount: u128, //should be u256
        // Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
        normalizedArbiterFee: u128, // should be u256
        // Portion of msg.value to be paid as the core bridge fee
        wormholeFee: u128,
    }

    struct AssetMeta<phantom CoinType> has key, store {
        // PayloadID uint8 = 2
        payloadID: u8,
        // Address of the token. Left-zero-padded if shorter than 32 bytes
        tokenAddress: vector<u8>,
        // Chain ID of the token
        tokenChain: u64,
        // Number of decimals of the token (big-endian uint256)
        decimals: u8,
        // Symbol of the token (UTF-8)
        symbol: vector<u8>,
        // Name of the token (UTF-8)
        name: vector<u8>,
    }
    
    struct RegisterChain has key{
        // Governance Header
        // module: "TokenBridge" left-padded
        mod: vector<u8>, //note: module keyword is reserved in Move
        // governance action: 1
        action: u8,
        // governance paket chain id: this or 0
        chainId: u64,//should be u16

        // Chain ID
        emitterChainID: u64, //should be u16
        // Emitter address. Left-zero-padded if shorter than 32 bytes
        emitterAddress: vector<u8>,
    }

    struct UpgradeContract has key{
        // Governance Header
        // module: "TokenBridge" left-padded
        mod: vector<u8>, //note: module keyword is reserved in Move
        // governance action: 2
        action: u8,
        // governance paket chain id
        chainId: u64,//should be u16

        // Address of the new contract
        newContract: vector<u8>,
    }
}