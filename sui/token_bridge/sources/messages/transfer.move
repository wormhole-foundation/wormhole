module token_bridge::transfer {
    use std::vector::{Self};
    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{Self, ExternalAddress};

    use token_bridge::normalized_amount::{Self, NormalizedAmount};

    friend token_bridge::transfer_tokens;

    #[test_only]
    friend token_bridge::complete_transfer_test;
    #[test_only]
    friend token_bridge::transfer_test;

    const E_INVALID_ACTION: u64 = 0;

    const PAYLOAD_ID: u8 = 1;

    struct Transfer has drop {
        // Amount being transferred.
        amount: NormalizedAmount,
        // Address of the token. Left-zero-padded if shorter than 32 bytes.
        token_address: ExternalAddress,
        // Chain ID of the token.
        token_chain: u16,
        // Address of the recipient. Left-zero-padded if shorter than 32 bytes.
        recipient: ExternalAddress,
        // Chain ID of the recipient.
        recipient_chain: u16,
        // Amount of tokens that the user is willing to pay as relayer fee.
        // Must be <= amount.
        relayer_fee: NormalizedAmount,
    }

    public fun new(
        amount: NormalizedAmount,
        token_address: ExternalAddress,
        token_chain: u16,
        recipient: ExternalAddress,
        recipient_chain: u16,
        relayer_fee: NormalizedAmount,
    ): Transfer {
        Transfer {
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            relayer_fee,
        }
    }

    public fun amount(self: &Transfer): NormalizedAmount {
        self.amount
    }

    public fun token_address(self: &Transfer): ExternalAddress {
        self.token_address
    }

    public fun token_chain(self: &Transfer): u16 {
        self.token_chain
    }

    public fun recipient(self: &Transfer): ExternalAddress {
        self.recipient
    }

    public fun recipient_chain(self: &Transfer): u16 {
        self.recipient_chain
    }

    public fun relayer_fee(self: &Transfer): NormalizedAmount {
        self.relayer_fee
    }

    public fun deserialize(buf: vector<u8>): Transfer {
        let cur = cursor::new(buf);
        assert!(
            bytes::deserialize_u8(&mut cur) == PAYLOAD_ID,
            E_INVALID_ACTION
        );
        let amount = normalized_amount::deserialize_be(&mut cur);
        let token_address = external_address::deserialize(&mut cur);
        let token_chain = bytes::deserialize_u16_be(&mut cur);
        let recipient = external_address::deserialize(&mut cur);
        let recipient_chain = bytes::deserialize_u16_be(&mut cur);
        let relayer_fee = normalized_amount::deserialize_be(&mut cur);
        cursor::destroy_empty(cur);
        new(
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            relayer_fee,
        )
    }

    public fun serialize(transfer: Transfer): vector<u8> {
        let buf = vector::empty<u8>();
        bytes::serialize_u8(&mut buf, PAYLOAD_ID);
        normalized_amount::serialize_be(&mut buf, transfer.amount);
        external_address::serialize(&mut buf, transfer.token_address);
        bytes::serialize_u16_be(&mut buf, transfer.token_chain);
        external_address::serialize(&mut buf, transfer.recipient);
        bytes::serialize_u16_be(&mut buf, transfer.recipient_chain);
        normalized_amount::serialize_be(&mut buf, transfer.relayer_fee);
        buf
    }

}

#[test_only]
module token_bridge::transfer_test {
    use token_bridge::transfer;
    use token_bridge::normalized_amount;
    use wormhole::external_address;

    #[test]
    public fun parse_roundtrip() {
        let amount = normalized_amount::from_raw(100, 8);
        let token_address = external_address::from_bytes(x"beef");
        let token_chain = 1;
        let recipient = external_address::from_bytes(x"cafe");
        let recipient_chain = 7;
        let fee = normalized_amount::from_raw(50, 8);
        let transfer = transfer::new(
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            fee,
        );
        let transfer = transfer::deserialize(transfer::serialize(transfer));
        assert!(transfer::amount(&transfer) == amount, 0);
        assert!(transfer::token_address(&transfer) == token_address, 0);
        assert!(transfer::token_chain(&transfer) == token_chain, 0);
        assert!(transfer::recipient(&transfer) == recipient, 0);
        assert!(transfer::recipient_chain(&transfer) == recipient_chain, 0);
        assert!(transfer::relayer_fee(&transfer) == fee, 0);
    }
}
