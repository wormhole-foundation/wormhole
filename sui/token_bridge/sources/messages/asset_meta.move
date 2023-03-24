module token_bridge::asset_meta {
    use std::string::{String};
    use std::vector::{Self};
    use wormhole::bytes::{Self};
    use wormhole::bytes32::{Self};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::cursor::{Self};

    const E_INVALID_ACTION: u64 = 0;

    const PAYLOAD_ID: u8 = 2;

    struct AssetMeta<phantom C> has copy, store, drop {
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

    public fun new<C>(
        token_address: ExternalAddress,
        token_chain: u16,
        native_decimals: u8,
        symbol: String,
        name: String,
    ): AssetMeta<C> {
        AssetMeta {
            token_address,
            token_chain,
            native_decimals,
            symbol,
            name
        }
    }

    public fun unpack<C>(
        meta: AssetMeta<C>
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

    public fun token_chain<C>(self: &AssetMeta<C>): u16 {
        self.token_chain
    }

    public fun token_address<C>(self: &AssetMeta<C>): ExternalAddress {
        self.token_address
    }

    public fun native_decimals<C>(self: &AssetMeta<C>): u8 {
        self.native_decimals
    }

    public fun symbol<C>(self: &AssetMeta<C>): String {
        self.symbol
    }

    public fun name<C>(self: &AssetMeta<C>): String {
        self.name
    }

    public fun serialize<C>(meta: AssetMeta<C>): vector<u8> {
        let buf = vector::empty<u8>();
        bytes::push_u8(&mut buf, PAYLOAD_ID);
        vector::append(
            &mut buf,
            external_address::to_bytes(meta.token_address)
        );
        bytes::push_u16_be(&mut buf, meta.token_chain);
        bytes::push_u8(&mut buf, meta.native_decimals);
        vector::append(
            &mut buf,
            bytes32::to_bytes(bytes32::from_string(meta.symbol))
        );
        vector::append(
            &mut buf,
            bytes32::to_bytes(bytes32::from_string(meta.name))
        );

        buf
    }

    public fun deserialize<C>(buf: vector<u8>): AssetMeta<C> {
        let cur = cursor::new(buf);
        assert!(
            bytes::take_u8(&mut cur) == PAYLOAD_ID,
            E_INVALID_ACTION
        );
        let token_address =
            external_address::new(bytes32::take_bytes(&mut cur));
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

}

#[test_only]
module token_bridge::asset_meta_test {
    use std::string::{Self};
    use wormhole::external_address::{Self};

    use token_bridge::asset_meta::{Self};

    struct DUMMY {}

    #[test]
    fun test_asset_meta(){
        let token_address = external_address::from_any_bytes(x"001122");
        let symbol = string::utf8(b"a creative symbol");
        let name = string::utf8(b"a creative name");
        let asset_meta = asset_meta::new<DUMMY>(
            token_address, //token address
            3, // token chain
            4, //native decimals
            symbol, // symbol
            name, // name
        );
        // Serialize and deserialize TransferWithPayload object.
        let se = asset_meta::serialize(asset_meta);
        let de = asset_meta::deserialize<DUMMY>(se);

        // Test that the object fields are unchanged.
        assert!(asset_meta::token_chain(&de) == 3, 0);
        assert!(asset_meta::token_address(&de) == token_address, 0);
        assert!(asset_meta::native_decimals(&de) == 4, 0);
        assert!(asset_meta::symbol(&de) ==  symbol, 0);
        assert!(asset_meta::name(&de) == name, 0);
    }
}
