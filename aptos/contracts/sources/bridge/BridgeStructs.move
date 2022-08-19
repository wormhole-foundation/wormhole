// module Wormhole::BridgeStructs {
//     use 0x1::vector::{Self};
//     use Wormhole::Serialize::{Self, serialize_u8, serialize_u64, serialize_u128, serialize_vector};
//     use Wormhole::Deserialize::{Self, deserialize_u8, deserialize_u64, deserialize_u128, deserialize_vector};

//     struct Transfer<phantom CoinType> has key, store, drop{
//         // PayloadID uint8 = 1
//         payloadID: u8,
//         // Amount being transferred (big-endian uint256)
//         amount: u128,
//         // Address of the token. Left-zero-padded if shorter than 32 bytes
//         tokenAddress: vector<u8>,
//         // Chain ID of the token
//         tokenChain: u64,//should be u16
//         // Address of the recipient. Left-zero-padded if shorter than 32 bytes
//         to: vector<u8>,
//         // Chain ID of the recipient
//         toChain: u64,
//         // Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
//         fee: u128, //should be u256
//     }

//     struct TransferWithPayload<phantom CoinType> has key, store, drop {
//         // PayloadID uint8 = 3
//         payloadID: u8,
//         // Amount being transferred (big-endian uint256)
//         amount: u128, //should be u256
//         // Address of the token. Left-zero-padded if shorter than 32 bytes
//         tokenAddress: vector<u8>,
//         // Chain ID of the token
//         tokenChain: u64,
//         // Address of the recipient. Left-zero-padded if shorter than 32 bytes
//         to: vector<u8>,
//         // Chain ID of the recipient
//         toChain: u64, //should be u16
//         // Address of the message sender. Left-zero-padded if shorter than 32 bytes
//         fromAddress: vector<u8>,
//         // An arbitrary payload
//         payload: vector<u8>,
//     }

//     struct TransferResult<phantom CoinType> has key, store, drop {
//         // Chain ID of the token
//         tokenChain: u64, // should be u16
//         // Address of the token. Left-zero-padded if shorter than 32 bytes
//         tokenAddress: vector<u8>, 
//         // Amount being transferred (big-endian uint256)
//         normalizedAmount: u128, //should be u256
//         // Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
//         normalizedArbiterFee: u128, // should be u256
//         // Portion of msg.value to be paid as the core bridge fee
//         wormholeFee: u128,
//     }

//     struct AssetMeta<phantom CoinType> has key, store, drop {
//         // PayloadID uint8 = 2
//         payloadID: u8,
//         // Address of the token. Left-zero-padded if shorter than 32 bytes
//         tokenAddress: vector<u8>,
//         // Chain ID of the token
//         tokenChain: u64,
//         // Number of decimals of the token (big-endian uint256)
//         decimals: u8,
//         // Symbol of the token (UTF-8)
//         symbol: vector<u8>,
//         // Name of the token (UTF-8)
//         name: vector<u8>,
//     }
    
//     struct RegisterChain has key, store, drop{
//         // Governance Header
//         // module: "TokenBridge" left-padded
//         mod: vector<u8>, //note: module keyword is reserved in Move
//         // governance action: 1
//         action: u8,
//         // governance paket chain id: this or 0
//         chainId: u64,//should be u16

//         // Chain ID
//         emitterChainID: u64, //should be u16
//         // Emitter address. Left-zero-padded if shorter than 32 bytes
//         emitterAddress: vector<u8>,
//     }

//     struct UpgradeContract has key, store, drop{
//         // Governance Header
//         // module: "TokenBridge" left-padded
//         mod: vector<u8>, //note: module keyword is reserved in Move
//         // governance action: 2
//         action: u8,
//         // governance paket chain id
//         chainId: u64,//should be u16

//         // Address of the new contract
//         newContract: vector<u8>,
//     }

//     public fun encodeAssetMeta<T>(meta: AssetMeta<T>): vector<u8> {
//         let encoded = vector::empty<u8>();
//         serialize_u8(&mut encoded, meta.payloadID); 
//         serialize_vector(&mut encoded, meta.tokenAddress); //len=32
//         serialize_u64(&mut encoded, meta.tokenChain);
//         serialize_u8(&mut encoded, meta.decimals);
//         //serialize_u64(&mut encoded, vector::length<u8>(&meta.symbol)); 
//         serialize_vector(&mut encoded, meta.symbol);
//         //serialize_u64(&mut encoded, vector::length<u8>(&meta.name));
//         serialize_vector(&mut encoded, meta.name);  
//         encoded
//     }

//     public fun encodeTransfer<T>(transfer: Transfer<T>): vector<u8>{
//         let encoded = vector::empty<u8>();
//         serialize_u8(&mut encoded, transfer.payloadID); 
//         serialize_u128(&mut encoded, transfer.amount);
//         serialize_vector(&mut encoded, transfer.tokenAddress); //len=32
//         serialize_u64(&mut encoded, transfer.tokenChain);
//         serialize_vector(&mut encoded, transfer.to); //len=32
//         serialize_u64(&mut encoded, transfer.toChain);
//         serialize_u128(&mut encoded, transfer.fee); //TODO: make sure this is big-endian u256
//         encoded
//     }

//     public fun encodeTransferWithPayload<T>(transfer: TransferWithPayload<T>): vector<u8>{
//         let encoded = vector::empty<u8>();
//         serialize_u8(&mut encoded, transfer.payloadID); 
//         serialize_u128(&mut encoded, transfer.amount);
//         serialize_vector(&mut encoded, transfer.tokenAddress); //len=32
//         serialize_u64(&mut encoded, transfer.tokenChain);
//         serialize_vector(&mut encoded, transfer.to); //len=32
//         serialize_u64(&mut encoded, transfer.toChain);
//         serialize_vector(&mut encoded, transfer.fromAddress); //len=32
//         serialize_vector(&mut encoded, transfer.payload);
//         encoded
//     }

//     public fun parseAssetMeta<T>(meta: vector<u8>): AssetMeta<T>{
//         let (payloadID, transfer) = deserialize_u8(meta); 
//         let (tokenAddress, transfer) = deserialize_vector(meta, 32);
//         let (tokenChain, transfer) = deserialize_u64(meta);
//         let (decimals, transfer) = deserialize_u8(meta);
//         let (symbol,transfer) = deserialize_vector(meta, 32);
//         let (name, transfer) = deserialize_vector(meta, 32);  
//         AssetMeta<T> {
//             payloadID,
//             tokenAddress,
//             tokenChain,
//             decimals, 
//             symbol, 
//             name
//         }
//     }

//     public fun parseTransfer<T>(transfer: vector<u8>): Transfer<T>{
//         let (payloadID, transfer) = deserialize_u8(transfer); 
//         let (amount, transfer) = deserialize_u128(transfer);
//         let (tokenAddress, transfer) = deserialize_vector(transfer, 32);
//         let (tokenChain, transfer) = deserialize_u64(transfer);
//         let (to, transfer) = deserialize_vector(transfer, 32);
//         let (toChain, transfer) = deserialize_u64(transfer);
//         let (fee, transfer) = deserialize_u128(transfer);
//         Transfer<T> { 
//             payloadID, 
//             amount, 
//             tokenAddress,
//             tokenChain,
//             to,
//             toChain,
//             fee
//         }
//     }

//     public fun parseTransferWithPayload<T>(transfer: vector<u8>): TransferWithPayload<T>{
//         let (payloadID, transfer) = deserialize_u8(transfer); 
//         let (amount, transfer) = deserialize_u128(transfer);
//         let (tokenAddress, transfer) = deserialize_vector(transfer, 32);
//         let (tokenChain, transfer) = deserialize_u64(transfer);
//         let (to, transfer) = deserialize_vector(transfer, 32);
//         let (toChain, transfer) = deserialize_u64(transfer);
//         let (fromAddress, transfer) = deserialize_vector(transfer, 32);
//         let (payload, transfer) = deserialize_vector(transfer, vector::length<u8>(&transfer));
//         TransferWithPayload<T> {
//             payloadID,
//             amount,
//             tokenAddress,
//             tokenChain, 
//             to, 
//             toChain,
//             fromAddress,
//             payload
//         }
//     }
// }