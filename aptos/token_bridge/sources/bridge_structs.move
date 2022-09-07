module token_bridge::bridge_structs {
    use 0x1::vector::{Self};
    use wormhole::serialize::{serialize_u8, serialize_u16, serialize_u256, serialize_vector};
    use wormhole::deserialize::{deserialize_u8, deserialize_u16, deserialize_u256, deserialize_vector};
    use wormhole::cursor::{Self};

    use wormhole::u256::{U256};
    use wormhole::u16::{U16};

    friend token_bridge::bridge_implementation;
    friend token_bridge::bridge_state;

    struct Transfer has key, store, drop{
        // TODO: is there a need to store the payload id in the parsed type?
        // It's there in the wire format to instruct which type to deserialise
        // into, but here the type implicitly encodes that information already.
        // PayloadID uint8 = 1
        payload_id: u8,
        // Amount being transferred (big-endian uint256)
        amount: U256,
        // Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: vector<u8>,
        // Chain ID of the token
        token_chain: U16,//should be u16
        // Address of the recipient. Left-zero-padded if shorter than 32 bytes
        to: vector<u8>,
        // Chain ID of the recipient
        to_chain: U16,
        // Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
        fee: U256, //should be u256
    }
    public(friend) fun create_transfer(
        payload_id: u8,
        amount: U256,
        token_address: vector<u8>,
        token_chain: U16,
        to: vector<u8>,
        to_chain: U16,
        fee: U256,
    ): Transfer {
        Transfer {
            payload_id,
            amount,
            token_address,
            token_chain,
            to,
            to_chain,
            fee,
        }
    }


    struct TransferWithPayload has key, store, drop {
        // PayloadID uint8 = 3
        payload_id: u8,
        // Amount being transferred (big-endian uint256)
        amount: U256, //should be u256
        // Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: vector<u8>,
        // Chain ID of the token
        token_chain: U16,
        // Address of the recipient. Left-zero-padded if shorter than 32 bytes
        to: vector<u8>,
        // Chain ID of the recipient
        to_chain: U16, //should be u16
        // Address of the message sender. Left-zero-padded if shorter than 32 bytes
        from_address: vector<u8>,
        // An arbitrary payload
        payload: vector<u8>,
    }

    struct TransferResult has key, store, drop {
        // Chain ID of the token
        token_chain: U16,
        // Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: vector<u8>,
        // Amount being transferred (big-endian uint256)
        normalized_amount: U256,
        // Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
        normalized_relayer_fee: U256, // should be u256
        // Portion of msg.value to be paid as the core bridge fee
        wormhole_fee: U256,
    }

    public(friend) fun create_transfer_result(
        token_chain: U16,
        token_address: vector<u8>,
        normalized_amount: U256,
        normalized_relayer_fee: U256,
        wormhole_fee: U256,
        ): TransferResult {
            TransferResult {
                token_chain,
                token_address,
                normalized_amount,
                normalized_relayer_fee,
                wormhole_fee
            }
    }

    struct AssetMeta has key, store, drop {
        // PayloadID uint8 = 2
        payload_id: u8,
        // Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: vector<u8>,
        // Chain ID of the token
        token_chain: U16,
        // Number of decimals of the token (big-endian uint256)
        decimals: u8,
        // Symbol of the token (UTF-8)
        symbol: vector<u8>,
        // Name of the token (UTF-8)
        name: vector<u8>,
    }

    public fun get_payload_id(a: &AssetMeta): u8 {
        a.payload_id
    }

    public fun get_token_address(a: &AssetMeta): vector<u8> {
        a.token_address
    }

    public fun get_token_chain(a: &AssetMeta): U16 {
        a.token_chain
    }

    public fun get_decimals(a: &AssetMeta): u8 {
        a.decimals
    }

    public fun get_symbol(a: &AssetMeta): vector<u8> {
        a.symbol
    }

    public fun get_name(a: &AssetMeta): vector<u8> {
        a.name
    }

    struct RegisterChain has key, store, drop{
        // TODO: same as above -- we shouldn't keep this in the parsed type,
        // only check in the parser.
        // Governance Header
        // module: "TokenBridge" left-padded
        mod: vector<u8>, //note: module keyword is reserved in Move
        // governance action: 1
        // TODO: same; remove
        action: u8,
        // governance paket chain id: this or 0
        chain_id: U16,

        // Chain ID
        emitter_chain_id: U16,
        // Emitter address. Left-zero-padded if shorter than 32 bytes
        emitter_address: vector<u8>,
    }

    struct UpgradeContract has key, store, drop{
        // Governance Header
        // module: "TokenBridge" left-padded
        mod: vector<u8>, //note: module keyword is reserved in Move
        // governance action: 2
        action: u8,
        // governance packet chain id
        chain_id: U16,

        // Address of the new contract
        new_contract: vector<u8>,
    }

    public fun create_asset_meta(
        payload_id: u8,
        // Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: vector<u8>,
        // Chain ID of the token
        token_chain: U16,
        // Number of decimals of the token (big-endian uint256)
        decimals: u8,
        // Symbol of the token (UTF-8)
        // TODO: symbol and name need to be padded (or truncated) to 32 bytes we
        // should introduce a custom type for this to make it more explicit
        // (something like String32). This applies to all vectors that are fixed
        // length, and we should only use `serialize_vector` for fields that
        // genuinely have a dynamic length (like the payload). Serialising
        // potentially un-validated data into what we expect to be a fixed
        // number of bytes is a recipe for disaster.
        symbol: vector<u8>,
        // Name of the token (UTF-8)
        name: vector<u8>,
    ): AssetMeta {
        AssetMeta{
            payload_id,
            token_address,
            token_chain,
            decimals,
            symbol,
            name
        }
    }

    // Construct a seed using AssetMeta fields for creating a new resource account 
    // N.B. seed is product of coin native chain and native address
    public(friend) fun create_seed(asset_meta: &AssetMeta): vector<u8>{
        let token_chain = get_token_chain(asset_meta);
        let token_address = get_token_address(asset_meta);
        let seed = vector::empty<u8>();
        serialize_u16(&mut seed, token_chain);
        serialize_vector(&mut seed, b"::");
        serialize_vector(&mut seed, token_address);
        seed
    }

    public fun encode_asset_meta(meta: AssetMeta): vector<u8> {
        let encoded = vector::empty<u8>();
        serialize_u8(&mut encoded, meta.payload_id);
        serialize_vector(&mut encoded, meta.token_address);
        serialize_u16(&mut encoded, meta.token_chain);
        serialize_u8(&mut encoded, meta.decimals);
        serialize_vector(&mut encoded, meta.symbol);
        serialize_vector(&mut encoded, meta.name);
        encoded
    }

    public fun encode_transfer(transfer: Transfer): vector<u8> {
        let encoded = vector::empty<u8>();
        serialize_u8(&mut encoded, transfer.payload_id);
        serialize_u256(&mut encoded, transfer.amount);
        serialize_vector(&mut encoded, transfer.token_address);
        serialize_u16(&mut encoded, transfer.token_chain);
        serialize_vector(&mut encoded, transfer.to);
        serialize_u16(&mut encoded, transfer.to_chain);
        serialize_u256(&mut encoded, transfer.fee);
        encoded
    }

    public fun encode_transfer_with_payload(transfer: TransferWithPayload): vector<u8> {
        let encoded = vector::empty<u8>();
        serialize_u8(&mut encoded, transfer.payload_id);
        serialize_u256(&mut encoded, transfer.amount);
        serialize_vector(&mut encoded, transfer.token_address);
        serialize_u16(&mut encoded, transfer.token_chain);
        serialize_vector(&mut encoded, transfer.to);
        serialize_u16(&mut encoded, transfer.to_chain);
        serialize_vector(&mut encoded, transfer.from_address);
        serialize_vector(&mut encoded, transfer.payload);
        encoded
    }

    public fun parse_asset_meta(meta: vector<u8>): AssetMeta {
        let cur = cursor::init(meta);
        let payload_id = deserialize_u8(&mut cur);
        let token_address = deserialize_vector(&mut cur, 32);
        let token_chain = deserialize_u16(&mut cur);
        let decimals = deserialize_u8(&mut cur);
        let symbol = deserialize_vector(&mut cur, 32);
        let name = deserialize_vector(&mut cur, 32);
        cursor::destroy_empty(cur);
        AssetMeta {
            payload_id,
            token_address,
            token_chain,
            decimals,
            symbol,
            name
        }
    }

    public fun parse_transfer(transfer: vector<u8>): Transfer {
        let cur = cursor::init(transfer);
        let payload_id = deserialize_u8(&mut cur);
        let amount = deserialize_u256(&mut cur);
        let token_address = deserialize_vector(&mut cur, 32);
        let token_chain = deserialize_u16(&mut cur);
        let to = deserialize_vector(&mut cur, 32);
        let to_chain= deserialize_u16(&mut cur);
        let fee = deserialize_u256(&mut cur);
        cursor::destroy_empty(cur);
        Transfer {
            payload_id,
            amount,
            token_address,
            token_chain,
            to,
            to_chain,
            fee
        }
    }

    public fun parse_transfer_with_payload(transfer: vector<u8>): TransferWithPayload {
        let cur = cursor::init(transfer);
        let payload_id = deserialize_u8(&mut cur);
        let amount = deserialize_u256(&mut cur);
        let token_address = deserialize_vector(&mut cur, 32);
        let token_chain = deserialize_u16(&mut cur);
        let to = deserialize_vector(&mut cur, 32);
        let to_chain = deserialize_u16(&mut cur);
        let from_address = deserialize_vector(&mut cur, 32);
        let payload = cursor::rest(cur);
        TransferWithPayload {
            payload_id,
            amount,
            token_address,
            token_chain,
            to,
            to_chain,
            from_address,
            payload
        }
    }
}
