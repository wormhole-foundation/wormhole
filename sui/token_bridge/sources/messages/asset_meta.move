module token_bridge::asset_meta {
    use std::string::{String};
    use std::vector::{Self};
    use wormhole::bytes::{Self};
    use wormhole::bytes32::{Self};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::cursor::{Self};

    const E_INVALID_ACTION: u64 = 0;

    const PAYLOAD_ID: u8 = 2;

    struct AssetMeta has copy, store, drop {
        /// Address of the token.
        token_address: ExternalAddress,
        /// Chain ID of the token.
        token_chain: u16,
        /// Number of decimals of the token.
        native_decimals: u8,
        /// Symbol of the token (UTF-8).
        symbol: String,
        /// Name of the token (UTF-8).
        name: String,
    }

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

    public fun unpack(
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

    public fun token_chain(self: &AssetMeta): u16 {
        self.token_chain
    }

    public fun token_address(self: &AssetMeta): ExternalAddress {
        self.token_address
    }

    public fun serialize(meta: AssetMeta): vector<u8> {
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
            bytes32::to_bytes(bytes32::from_string(symbol))
        );
        vector::append(
            &mut buf,
            bytes32::to_bytes(bytes32::from_string(name))
        );

        buf
    }

    public fun deserialize(buf: vector<u8>): AssetMeta {
        let cur = cursor::new(buf);
        assert!(bytes::take_u8(&mut cur) == PAYLOAD_ID, E_INVALID_ACTION);
        let token_address = external_address::take_bytes(&mut cur);
        let token_chain = bytes::take_u16_be(&mut cur);
        let native_decimals = bytes::take_u8(&mut cur);
        let symbol = bytes32::to_string(bytes32::take_bytes(&mut cur));
        let name = bytes32::to_string(bytes32::take_bytes(&mut cur));
        cursor::destroy_empty(cur);

        new(
            token_address,
            token_chain,
            native_decimals,
            symbol,
            name
        )
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

}

#[test_only]
module token_bridge::asset_meta_tests {
    use std::string::{Self};
    use wormhole::external_address::{Self};

    use token_bridge::asset_meta::{Self};

    #[test]
    fun test_asset_meta() {
        let token_address = external_address::from_any_bytes(x"001122");
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
        let se = asset_meta::serialize(asset_meta);
        let de = asset_meta::deserialize(se);

        // Test that the object fields are unchanged.
        assert!(asset_meta::token_chain(&de) == 3, 0);
        assert!(asset_meta::token_address(&de) == token_address, 0);
        assert!(asset_meta::native_decimals(&de) == 4, 0);
        assert!(asset_meta::symbol(&de) ==  symbol, 0);
        assert!(asset_meta::name(&de) == name, 0);
    }
}
