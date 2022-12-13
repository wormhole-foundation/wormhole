module nft_bridge::transfer {
    use std::vector;
    use wormhole::serialize::{
        serialize_u8,
        serialize_u16,
    };
    use wormhole::deserialize::{
        deserialize_u8,
        deserialize_u16,
    };
    use wormhole::cursor;
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::u16::U16;

    use nft_bridge::uri::{Self, URI};

    use token_bridge::string32::{Self, String32};

    friend nft_bridge::transfer_nft;

    #[test_only]
    friend nft_bridge::complete_transfer_test;
    #[test_only]
    friend nft_bridge::transfer_test;

    const E_INVALID_ACTION: u64 = 0;

    struct Transfer has drop {
        /// Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: ExternalAddress,
        /// Chain ID of the token
        token_chain: U16,
        /// Symbol of the token
        symbol: String32,
        /// Name of the token
        name: String32,
        /// Token ID
        token_id: ExternalAddress,
        /// URI of the token metadata
        uri: URI,
        /// Address of the recipient. Left-zero-padded if shorter than 32 bytes
        to: ExternalAddress,
        /// Chain ID of the recipient
        to_chain: U16,
    }

    public fun get_token_address(a: &Transfer): ExternalAddress {
        a.token_address
    }

    public fun get_token_chain(a: &Transfer): U16 {
        a.token_chain
    }

    public fun get_symbol(a: &Transfer): String32 {
        a.symbol
    }

    public fun get_name(a: &Transfer): String32 {
        a.name
    }

    public fun get_token_id(a: &Transfer): ExternalAddress {
        a.token_id
    }

    public fun get_uri(a: &Transfer): URI {
        a.uri
    }

    public fun get_to(a: &Transfer): ExternalAddress {
        a.to
    }

    public fun get_to_chain(a: &Transfer): U16 {
        a.to_chain
    }

    public(friend) fun create(
        token_address: ExternalAddress,
        token_chain: U16,
        symbol: String32,
        name: String32,
        token_id: ExternalAddress,
        uri: URI,
        to: ExternalAddress,
        to_chain: U16,
    ): Transfer {
        Transfer {
            token_address,
            token_chain,
            symbol,
            name,
            token_id,
            uri,
            to,
            to_chain,
        }
    }

    public fun parse(transfer: vector<u8>): Transfer {
        let cur = cursor::init(transfer);
        let action = deserialize_u8(&mut cur);
        assert!(action == 1, E_INVALID_ACTION);
        let token_address = external_address::deserialize(&mut cur);
        let token_chain = deserialize_u16(&mut cur);
        let symbol = string32::deserialize(&mut cur);
        let name = string32::deserialize(&mut cur);
        let token_id = external_address::deserialize(&mut cur);
        let uri = uri::deserialize(&mut cur);
        let to = external_address::deserialize(&mut cur);
        let to_chain = deserialize_u16(&mut cur);
        cursor::destroy_empty(cur);
        Transfer {
            token_address,
            token_chain,
            symbol,
            name,
            token_id,
            uri,
            to,
            to_chain,
        }
    }

    public fun encode(transfer: Transfer): vector<u8> {
        let encoded = vector::empty<u8>();
        serialize_u8(&mut encoded, 1);
        external_address::serialize(&mut encoded, transfer.token_address);
        serialize_u16(&mut encoded, transfer.token_chain);
        string32::serialize(&mut encoded, transfer.symbol);
        string32::serialize(&mut encoded, transfer.name);
        external_address::serialize(&mut encoded, transfer.token_id);
        uri::serialize(&mut encoded, transfer.uri);
        external_address::serialize(&mut encoded, transfer.to);
        serialize_u16(&mut encoded, transfer.to_chain);
        encoded
    }

}

#[test_only]
module nft_bridge::transfer_test {
    use nft_bridge::transfer;
    use nft_bridge::uri;
    use token_bridge::string32;
    use wormhole::external_address;
    use wormhole::u16;

    #[test]
    public fun parse_roundtrip() {
        let token_address = external_address::from_bytes(x"beef");
        let token_chain = u16::from_u64(1);
        let symbol = string32::from_bytes(b"HELLO");
        let name = string32::from_bytes(b"hello token");
        let to = external_address::from_bytes(x"cafe");
        let token_id = external_address::from_bytes(x"beefcafe");
        let uri = uri::from_bytes(b"http://google.com");
        let to_chain = u16::from_u64(7);
        let transfer = transfer::create(
            token_address,
            token_chain,
            symbol,
            name,
            token_id,
            uri,
            to,
            to_chain,
        );
        let transfer = transfer::parse(transfer::encode(transfer));
        assert!(transfer::get_token_address(&transfer) == token_address, 0);
        assert!(transfer::get_token_chain(&transfer) == token_chain, 0);
        assert!(transfer::get_symbol(&transfer) == symbol, 0);
        assert!(transfer::get_name(&transfer) == name, 0);
        assert!(transfer::get_token_id(&transfer) == token_id, 0);
        assert!(transfer::get_uri(&transfer) == uri, 0);
        assert!(transfer::get_to(&transfer) == to, 0);
        assert!(transfer::get_to_chain(&transfer) == to_chain, 0);
    }
}
