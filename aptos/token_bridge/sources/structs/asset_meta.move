module token_bridge::asset_meta {
    use 0x1::vector::{Self};
    use wormhole::serialize::{serialize_u8, serialize_u16, serialize_vector};
    use wormhole::deserialize::{deserialize_u8, deserialize_u16, deserialize_vector};
    use wormhole::cursor::{Self};

    use wormhole::u16::{U16};

    friend token_bridge::bridge_implementation;
    // friend token_bridge::bridge_state;

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

    public(friend) fun create(
        // TODO: delete payload_id
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

    public fun encode(meta: AssetMeta): vector<u8> {
        let encoded = vector::empty<u8>();
        serialize_u8(&mut encoded, meta.payload_id);
        serialize_vector(&mut encoded, meta.token_address);
        serialize_u16(&mut encoded, meta.token_chain);
        serialize_u8(&mut encoded, meta.decimals);
        serialize_vector(&mut encoded, meta.symbol);
        serialize_vector(&mut encoded, meta.name);
        encoded
    }

    // TODO: the parse functions should be private I think
    public fun parse(meta: vector<u8>): AssetMeta {
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

}
