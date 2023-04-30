// SPDX-License-Identifier: Apache 2

/// This module implements serialization and deserialization for asset metadata,
/// which is a specific Wormhole message payload for Token Bridge.
module token_bridge::asset_meta {
    use std::string::{Self, String};
    use std::vector::{Self};
    use sui::coin::{Self, CoinMetadata};
    use wormhole::bytes::{Self};
    use wormhole::bytes32::{Self};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::cursor::{Self};
    use wormhole::state::{chain_id};

    use token_bridge::native_asset::{Self};

    friend token_bridge::attest_token;
    friend token_bridge::create_wrapped;
    friend token_bridge::wrapped_asset;

    /// Message payload is not `AssetMeta`.
    const E_INVALID_PAYLOAD: u64 = 0;

    /// Message identifier.
    const PAYLOAD_ID: u8 = 2;

    /// Container that warehouses asset metadata information. This struct is
    /// used only by `attest_token` and `create_wrapped` modules.
    struct AssetMeta {
        /// Address of the token.
        token_address: ExternalAddress,
        /// Chain ID of the token.
        token_chain: u16,
        /// Number of decimals of the token.
        native_decimals: u8,
        /// Symbol of the token (UTF-8).
        /// TODO(csongor): maybe turn these into String32s?
        symbol: String,
        /// Name of the token (UTF-8).
        name: String,
    }


    public(friend) fun from_metadata<C>(metadata: &CoinMetadata<C>): AssetMeta {
        AssetMeta {
            token_address: native_asset::canonical_address(metadata),
            token_chain: chain_id(),
            native_decimals: coin::get_decimals(metadata),
            symbol: string::from_ascii(coin::get_symbol(metadata)),
            name: coin::get_name(metadata)
        }
    }

    #[test_only]
    public fun from_metadata_test_only<C>(metadata: &CoinMetadata<C>): AssetMeta {
        from_metadata(metadata)
    }

    public(friend) fun unpack(
        meta: AssetMeta
    ): (
        ExternalAddress,
        u16,
        u8,
        String,
        String
    ) {
        let AssetMeta {
            token_address,
            token_chain,
            native_decimals,
            symbol,
            name
        } = meta;

        (
            token_address,
            token_chain,
            native_decimals,
            symbol,
            name
        )
    }


    #[test_only]
    public fun unpack_test_only(
        meta: AssetMeta
    ): (
        ExternalAddress,
        u16,
        u8,
        String,
        String
    ) {
        unpack(meta)
    }

    public fun token_chain(self: &AssetMeta): u16 {
        self.token_chain
    }

    public fun token_address(self: &AssetMeta): ExternalAddress {
        self.token_address
    }

    public(friend) fun serialize(meta: AssetMeta): vector<u8> {
        let (
            token_address,
            token_chain,
            native_decimals,
            symbol,
            name
        ) = unpack(meta);

        let buf = vector::empty<u8>();
        bytes::push_u8(&mut buf, PAYLOAD_ID);
        vector::append(&mut buf, external_address::to_bytes(token_address));
        bytes::push_u16_be(&mut buf, token_chain);
        bytes::push_u8(&mut buf, native_decimals);
        vector::append(
            &mut buf,
            bytes32::to_bytes(bytes32::from_utf8(symbol))
        );
        vector::append(
            &mut buf,
            bytes32::to_bytes(bytes32::from_utf8(name))
        );

        buf
    }

    #[test_only]
    public fun serialize_test_only(meta: AssetMeta): vector<u8> {
        serialize(meta)
    }

    public(friend) fun deserialize(buf: vector<u8>): AssetMeta {
        let cur = cursor::new(buf);
        assert!(bytes::take_u8(&mut cur) == PAYLOAD_ID, E_INVALID_PAYLOAD);
        let token_address = external_address::take_bytes(&mut cur);
        let token_chain = bytes::take_u16_be(&mut cur);
        let native_decimals = bytes::take_u8(&mut cur);
        let symbol = bytes32::to_utf8(bytes32::take_bytes(&mut cur));
        let name = bytes32::to_utf8(bytes32::take_bytes(&mut cur));
        cursor::destroy_empty(cur);

        AssetMeta {
            token_address,
            token_chain,
            native_decimals,
            symbol,
            name
        }
    }

    #[test_only]
    public fun deserialize_test_only(buf: vector<u8>): AssetMeta {
        deserialize(buf)
    }

    #[test_only]
    public fun new(
        token_address: ExternalAddress,
        token_chain: u16,
        native_decimals: u8,
        symbol: String,
        name: String,
    ): AssetMeta {
        AssetMeta {
            token_address,
            token_chain,
            native_decimals,
            symbol,
            name
        }
    }

    #[test_only]
    public fun native_decimals(self: &AssetMeta): u8 {
        self.native_decimals
    }

    #[test_only]
    public fun symbol(self: &AssetMeta): String {
        self.symbol
    }

    #[test_only]
    public fun name(self: &AssetMeta): String {
        self.name
    }

    #[test_only]
    public fun destroy(token_meta: AssetMeta) {
        unpack(token_meta);
    }

    #[test_only]
    public fun payload_id(): u8 {
        PAYLOAD_ID
    }
}

#[test_only]
module token_bridge::asset_meta_tests {
    use std::string::{Self};
    use wormhole::external_address::{Self};
    use wormhole::vaa::{Self};

    use token_bridge::asset_meta::{Self};

    #[test]
    fun test_serialize_deserialize() {
        let token_address = external_address::from_address(@0x1122);
        let symbol = string::utf8(b"a creative symbol");
        let name = string::utf8(b"a creative name");
        let asset_meta = asset_meta::new(
            token_address, //token address
            3, // token chain
            4, //native decimals
            symbol, // symbol
            name, // name
        );
        // Serialize and deserialize TransferWithPayload object.
        let se = asset_meta::serialize_test_only(asset_meta);
        let de = asset_meta::deserialize_test_only(se);

        // Test that the object fields are unchanged.
        assert!(asset_meta::token_chain(&de) == 3, 0);
        assert!(asset_meta::token_address(&de) == token_address, 0);
        assert!(asset_meta::native_decimals(&de) == 4, 0);
        assert!(asset_meta::symbol(&de) ==  symbol, 0);
        assert!(asset_meta::name(&de) == name, 0);

        // Clean up.
        asset_meta::destroy(de);
    }

    #[test]
    fun test_create_wrapped_12() {
        use token_bridge::dummy_message::{encoded_asset_meta_vaa_foreign_12};

        let payload =
            vaa::peel_payload_from_vaa(&encoded_asset_meta_vaa_foreign_12());

        let token_meta = asset_meta::deserialize_test_only(payload);
        let serialized = asset_meta::serialize_test_only(token_meta);
        assert!(payload == serialized, 0);
    }
}
