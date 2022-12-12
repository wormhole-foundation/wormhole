module token_bridge::asset_meta {
    use std::vector::{Self};
    use wormhole::serialize::{serialize_u8, serialize_u16, serialize_vector};
    use wormhole::deserialize::{deserialize_u8, deserialize_u16, deserialize_vector};
    use wormhole::cursor::{Self};

    use wormhole::u16::{U16};
    use wormhole::external_address::{Self, ExternalAddress};

    use token_bridge::string32::{Self, String32};

    friend token_bridge::attest_token;
    friend token_bridge::wrapped;

    #[test_only]
    friend token_bridge::wrapped_test;

    const E_INVALID_ACTION: u64 = 0;

    struct AssetMeta has key, store, drop {
        /// Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: ExternalAddress,
        /// Chain ID of the token
        token_chain: U16,
        /// Number of decimals of the token (big-endian uint256)
        decimals: u8,
        /// Symbol of the token (UTF-8)
        symbol: String32,
        /// Name of the token (UTF-8)
        name: String32,
    }

    public fun get_token_address(a: &AssetMeta): ExternalAddress {
        a.token_address
    }

    public fun get_token_chain(a: &AssetMeta): U16 {
        a.token_chain
    }

    public fun get_decimals(a: &AssetMeta): u8 {
        a.decimals
    }

    public fun get_symbol(a: &AssetMeta): String32 {
        a.symbol
    }

    public fun get_name(a: &AssetMeta): String32 {
        a.name
    }

    public(friend) fun create(
        token_address: ExternalAddress,
        token_chain: U16,
        decimals: u8,
        symbol: String32,
        name: String32,
    ): AssetMeta {
        AssetMeta {
            token_address,
            token_chain,
            decimals,
            symbol,
            name
        }
    }

    public fun encode(meta: AssetMeta): vector<u8> {
        let encoded = vector::empty<u8>();
        serialize_u8(&mut encoded, 2);
        serialize_vector(&mut encoded, external_address::get_bytes(&meta.token_address));
        serialize_u16(&mut encoded, meta.token_chain);
        serialize_u8(&mut encoded, meta.decimals);
        string32::serialize(&mut encoded, meta.symbol);
        string32::serialize(&mut encoded, meta.name);
        encoded
    }

    public fun parse(meta: vector<u8>): AssetMeta {
        let cur = cursor::init(meta);
        let action = deserialize_u8(&mut cur);
        assert!(action == 2, E_INVALID_ACTION);
        let token_address = deserialize_vector(&mut cur, 32);
        let token_chain = deserialize_u16(&mut cur);
        let decimals = deserialize_u8(&mut cur);
        let symbol = string32::deserialize(&mut cur);
        let name = string32::deserialize(&mut cur);
        cursor::destroy_empty(cur);
        AssetMeta {
            token_address: external_address::from_bytes(token_address),
            token_chain,
            decimals,
            symbol,
            name
        }
    }

    // Construct a seed using AssetMeta fields for creating a new resource account
    // N.B. seed is a product of coin native chain and native address
    public(friend) fun create_seed(asset_meta: &AssetMeta): vector<u8> {
        let token_chain = get_token_chain(asset_meta);
        let token_address = get_token_address(asset_meta);
        let seed = vector::empty<u8>();
        serialize_u16(&mut seed, token_chain);
        serialize_vector(&mut seed, b"::");
        external_address::serialize(&mut seed, token_address);
        seed
    }

}
