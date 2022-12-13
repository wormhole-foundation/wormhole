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
    #[test_only]
    friend nft_bridge::wrapped_test;

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

    public fun encode(transfer: &Transfer): vector<u8> {
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

    // VAA from https://etherscan.io/tx/0x8250625a8dfb66ecc9d5b8fd188057ef332c0a1d09a1510131f7b104e8dbf79b
    const SOLANA_NFT: vector<u8> = x"01000000010d004ee8f9a0a898aedc2340289880ef662b55333cb4a9a374282f494706a780815e5462aff9e2f59da0a7402486ad5a5ea4e5604c4c3270c269dde700ff63011a7600022f1cd8622698fdbe116477e164175243271dc5a47a649fbbfc5ab44d7c0c8efc1ec9df05f7543dea0f67faa84ac53953bf57dd7a674dfe124fbc941e5471d5100103b63085d5a9fdc531d6e04f907fd1bd80217016878bf44fea1f41f5408a1e62426bf64e115b732aa53154f199f4299a5b1a4110ec3dbc1d69f4f4e386489da98e0104e46ac1fc4c35b8835191c9ec8f7146a212b33ce8592f8311addbcbed9fdd3caa257cb43cd07de39cc76f17edee89fb424f5797b08a71ab8a913b1c899d0182f3000561d3ba8f0ea84da5ccdf916ab22ed2c55b5ab2a2d0bfb8d6e4487285bb1817fb272b74aba27822045ad1b5dfd4378e8aadfef290c1c13007f7bd2f4f611cf32800070b83d82dd6dc975c1bcaefb8acb0e8127015805940d89366bf1d86988650e809522a29bf1c30393cf1bf46d5f0beefbe01dde1f20807116148e8dd20d339c24b00090f44858974ad211f9d21d7d59eb1b10207f7c1ae4591d3eaaef86afcc7afbe926584e38b58fc5d441890b905897346f1a8f9fca48217509a0415f06f80586804010a1de1bf4416235e41d56d5634161eb8544f3d1c82a5ee666c13402ad8652e706112af89e902974b61c4be8d253c24eb858872cf80d556bc8288487fd18c9673f8010c104c3c79c2224fa5bd04ab2737487cc2b49f05096237d3678ccbe57b92791471473e735c09659083758702ba241f99d6550a02b11cc43ba407c2eb1a294eb491010d436bc9540cef59afa60764c5d43640d854ff847f3afce0844c7de085e0e45df34693871393e701d15a7a5385c4c394c394b2d6427b7f16b57c89bd8d1e83a721000e56baf9369bcef85e1b705a7e5904f576101ff28dddce6078cfb1ee9ce95170c70ff05e6a8ff500b3a44877894fec08fd16244764675127c2497200301e76bb13010f1e8b4335050134d419ff985ba3d26f734eb19e78a0a3b39739a2eb9993eeb62c19569ac040f535e6f9093366fce418f127fe548dff25cba8d3b9b65f24306cc80010ae2a582ba0435d155115b6bfe9c92b5f7c0147670bd4b6d815f9507f689f80f7495114c0e742c108dc982038be28e89bde5124306b6cc39cbb3fd4ed32c74783006149baf70000cb2800010def15a24423e1edd1a5ab16f557b9060303ddbab8c803d2ee48f4b78a1cfd6b000000000000000001010101010101010101010101010101010101010101010101010101010101010101000154535400000000000000000000000000000000000000000000000000000000005465737400000000000000000000000000000000000000000000000000000000c4a79ff9105f87c3ed880ed4252b61759251acc6050c163c0a5d828d92dc0cb8c868747470733a2f2f6172742d73616e64626f782e73756e666c6f7765722e696e64757374726965732f746f6b656e2f3078643538343334663333613230363631663138366666363736323665613662646634316238306263612f393630000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000096d13cbeffe7bae169b9032fe69ed56eb07b300f0002";

    #[test]
    public fun parse_sol_transfer() {
        let vaa = wormhole::vaa::parse_test(SOLANA_NFT);
        let parsed_transfer = transfer::parse(wormhole::vaa::destroy(vaa));

        let token_address = external_address::from_bytes(x"0101010101010101010101010101010101010101010101010101010101010101");
        let token_chain = u16::from_u64(1);
        let symbol = string32::from_bytes(b"TST");
        let name = string32::from_bytes(b"Test");
        let to = external_address::from_bytes(x"00000000000000000000000096d13cbeffe7bae169b9032fe69ed56eb07b300f");
        let token_id = external_address::from_bytes(x"c4a79ff9105f87c3ed880ed4252b61759251acc6050c163c0a5d828d92dc0cb8");
        // The VAA uses all 200 characters and pads with a bunch of 0 bytes at the end
        let uri = uri::from_bytes(b"https://art-sandbox.sunflower.industries/token/0xd58434f33a20661f186ff67626ea6bdf41b80bca/960\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0");
        let to_chain = u16::from_u64(2);
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

        assert!(parsed_transfer == transfer, 0);
    }

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
        let parsed_transfer = transfer::parse(transfer::encode(&transfer));
        assert!(parsed_transfer == transfer, 0);
    }
}
