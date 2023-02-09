module token_bridge::transfer {
    use std::vector;
    use wormhole::bytes::{
        serialize_u8,
        serialize_u16_be,
    };
    use wormhole::bytes::{
        deserialize_u8,
        deserialize_u16_be,
    };
    use wormhole::cursor;
    use wormhole::external_address::{Self, ExternalAddress};

    use token_bridge::normalized_amount::{Self, NormalizedAmount};

    friend token_bridge::transfer_tokens;

    #[test_only]
    friend token_bridge::complete_transfer_test;
    #[test_only]
    friend token_bridge::transfer_test;

    const E_INVALID_ACTION: u64 = 0;

    struct Transfer has drop {
        /// Amount being transferred
        amount: NormalizedAmount,
        /// Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: ExternalAddress,
        /// Chain ID of the token
        token_chain: u16,
        /// Address of the recipient. Left-zero-padded if shorter than 32 bytes
        to: ExternalAddress,
        /// Chain ID of the recipient
        to_chain: u16,
        /// Amount of tokens that the user is willing to pay as relayer fee. Must be <= Amount.
        fee: NormalizedAmount,
    }

    public fun get_amount(a: &Transfer): NormalizedAmount {
        a.amount
    }

    public fun get_token_address(a: &Transfer): ExternalAddress {
        a.token_address
    }

    public fun get_token_chain(a: &Transfer): u16 {
        a.token_chain
    }

    public fun get_to(a: &Transfer): ExternalAddress {
        a.to
    }

    public fun get_to_chain(a: &Transfer): u16 {
        a.to_chain
    }

    public fun get_fee(a: &Transfer): NormalizedAmount {
        a.fee
    }

    public(friend) fun create(
        amount: NormalizedAmount,
        token_address: ExternalAddress,
        token_chain: u16,
        to: ExternalAddress,
        to_chain: u16,
        fee: NormalizedAmount,
    ): Transfer {
        Transfer {
            amount,
            token_address,
            token_chain,
            to,
            to_chain,
            fee,
        }
    }

    public fun parse(transfer: vector<u8>): Transfer {
        let cur = cursor::new(transfer);
        let action = deserialize_u8(&mut cur);
        assert!(action == 1, E_INVALID_ACTION);
        let amount = normalized_amount::deserialize_be(&mut cur);
        let token_address = external_address::deserialize(&mut cur);
        let token_chain = deserialize_u16_be(&mut cur);
        let to = external_address::deserialize(&mut cur);
        let to_chain = deserialize_u16_be(&mut cur);
        let fee = normalized_amount::deserialize_be(&mut cur);
        cursor::destroy_empty(cur);
        Transfer {
            amount,
            token_address,
            token_chain,
            to,
            to_chain,
            fee,
        }
    }

    public fun encode(transfer: Transfer): vector<u8> {
        let encoded = vector::empty<u8>();
        serialize_u8(&mut encoded, 1);
        normalized_amount::serialize_be(&mut encoded, transfer.amount);
        external_address::serialize(&mut encoded, transfer.token_address);
        serialize_u16_be(&mut encoded, transfer.token_chain);
        external_address::serialize(&mut encoded, transfer.to);
        serialize_u16_be(&mut encoded, transfer.to_chain);
        normalized_amount::serialize_be(&mut encoded, transfer.fee);
        encoded
    }

}

#[test_only]
module token_bridge::transfer_test {
    use token_bridge::transfer;
    use token_bridge::normalized_amount;
    use wormhole::external_address;

    #[test]
    public fun parse_roundtrip() {
        let amount = normalized_amount::normalize(100, 8);
        let token_address = external_address::from_bytes(x"beef");
        let token_chain: u16 = 1;
        let to = external_address::from_bytes(x"cafe");
        let to_chain: u16 = 7;
        let fee = normalized_amount::normalize(50, 8);
        let transfer = transfer::create(
            amount,
            token_address,
            token_chain,
            to,
            to_chain,
            fee,
        );
        let transfer = transfer::parse(transfer::encode(transfer));
        assert!(transfer::get_amount(&transfer) == amount, 0);
        assert!(transfer::get_token_address(&transfer) == token_address, 0);
        assert!(transfer::get_token_chain(&transfer) == token_chain, 0);
        assert!(transfer::get_to(&transfer) == to, 0);
        assert!(transfer::get_to_chain(&transfer) == to_chain, 0);
        assert!(transfer::get_fee(&transfer) == fee, 0);
    }
}
