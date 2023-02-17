module token_bridge::asset_meta {
    use std::string::{String};
    use std::vector::{Self};
    use wormhole::bytes::{Self};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::cursor::{Self};

    use token_bridge::string32::{Self, String32};

    friend token_bridge::state;
    friend token_bridge::create_wrapped;
    #[test_only]
    friend token_bridge::asset_meta_test;

    const E_INVALID_ACTION: u64 = 0;

    const PAYLOAD_ID: u8 = 2;

    struct AssetMeta has copy, store, drop {
        /// Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: ExternalAddress,
        /// Chain ID of the token
        token_chain: u16,
        /// Number of decimals of the token (big-endian uint256)
        native_decimals: u8,
        /// Symbol of the token (UTF-8)
        symbol: String32,
        /// Name of the token (UTF-8)
        name: String32,
    }

    public(friend) fun new(
        token_chain: u16,
        token_address: ExternalAddress,
        native_decimals: u8,
        symbol: String32,
        name: String32,
    ): AssetMeta {
        AssetMeta {
            token_address,
            token_chain,
            native_decimals,
            symbol,
            name
        }
    }

    public fun token_chain(self: &AssetMeta): u16 {
        self.token_chain
    }

    public fun token_address(self: &AssetMeta): ExternalAddress {
        self.token_address
    }

    public fun native_decimals(self: &AssetMeta): u8 {
        self.native_decimals
    }

    public fun symbol(self: &AssetMeta): String32 {
        self.symbol
    }

    public fun symbol_to_string(self: &AssetMeta): String {
        string32::to_string(&self.symbol)
    }

    public fun name(self: &AssetMeta): String32 {
        self.name
    }

    public fun name_to_string(self: &AssetMeta): String {
        string32::to_string(&self.name)
    }

    public fun serialize(meta: AssetMeta): vector<u8> {
        let buf = vector::empty<u8>();
        bytes::serialize_u8(&mut buf, PAYLOAD_ID);
        bytes::from_bytes(
            &mut buf,
            external_address::get_bytes(&meta.token_address)
        );
        bytes::serialize_u16_be(&mut buf, meta.token_chain);
        bytes::serialize_u8(&mut buf, meta.native_decimals);
        string32::serialize(&mut buf, meta.symbol);
        string32::serialize(&mut buf, meta.name);

        buf
    }

    public fun deserialize(buf: vector<u8>): AssetMeta {
        let cur = cursor::new(buf);
        assert!(
            bytes::deserialize_u8(&mut cur) == PAYLOAD_ID,
            E_INVALID_ACTION
        );
        let token_address =
            external_address::from_bytes(bytes::to_bytes(&mut cur, 32));
        let token_chain = bytes::deserialize_u16_be(&mut cur);
        let native_decimals = bytes::deserialize_u8(&mut cur);
        let symbol = string32::deserialize(&mut cur);
        let name = string32::deserialize(&mut cur);
        cursor::destroy_empty(cur);
        new(
            token_chain,
            token_address,
            native_decimals,
            symbol,
            name
        )
    }

}

#[test_only]
module token_bridge::asset_meta_test{
    use std::string::{Self};

    use wormhole::external_address::{Self};

    use token_bridge::asset_meta::{Self};
    use token_bridge::string32::{Self};

    #[test]
    fun test_asset_meta(){
        let symbol = string::utf8(b"a creative symbol");
        let name = string::utf8(b"a creative name");
        let token_address = external_address::from_bytes(x"001122");
        let symbol =  string32::from_string(&symbol);
        let name = string32::from_string(&name);
        let asset_meta = asset_meta::new(
            3, // token chain
            token_address, //token address
            4, //native decimals
            symbol, // symbol
            name, // name
        );
        // serialize and deserialize TransferWithPayload object
        let se = asset_meta::serialize(asset_meta);
        let de = asset_meta::deserialize(se);

        // test that the object fields are unchanged
        assert!(asset_meta::token_chain(&de) == 3, 0);
        assert!(asset_meta::token_address(&de) == token_address, 0);
        assert!(asset_meta::native_decimals(&de) == 4, 0);
        assert!(asset_meta::symbol(&de) ==  symbol, 0);
        assert!(asset_meta::name(&de) == name, 0);
    }
}
