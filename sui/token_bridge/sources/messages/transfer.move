module token_bridge::transfer {
    use std::vector::{Self};
    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{Self, ExternalAddress};

    use token_bridge::normalized_amount::{Self, NormalizedAmount};

    const E_INVALID_ACTION: u64 = 0;

    const PAYLOAD_ID: u8 = 1;

    struct Transfer {
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

    public fun payload_id(): u8 {
        PAYLOAD_ID
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

    public fun unpack(
        transfer: Transfer
    ): (
        NormalizedAmount,
        ExternalAddress,
        u16,
        ExternalAddress,
        u16,
        NormalizedAmount
    ) {
        let Transfer {
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            relayer_fee,
        } = transfer;

        (
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            relayer_fee
        )
    }

    public fun deserialize(buf: vector<u8>): Transfer {
        let cur = cursor::new(buf);
        assert!(bytes::take_u8(&mut cur) == PAYLOAD_ID, E_INVALID_ACTION);

        let amount = normalized_amount::take_bytes(&mut cur);
        let token_address = external_address::take_bytes(&mut cur);
        let token_chain = bytes::take_u16_be(&mut cur);
        let recipient = external_address::take_bytes(&mut cur);
        let recipient_chain = bytes::take_u16_be(&mut cur);
        let relayer_fee = normalized_amount::take_bytes(&mut cur);
        cursor::destroy_empty(cur);

        new(
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            relayer_fee
        )
    }

    public fun serialize(transfer: Transfer): vector<u8> {
        let (
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            relayer_fee,
        ) = unpack(transfer);

        let buf = vector::empty<u8>();
        bytes::push_u8(&mut buf, PAYLOAD_ID);
        bytes::push_u256_be(&mut buf, normalized_amount::to_u256(amount));
        vector::append(&mut buf, external_address::to_bytes(token_address));
        bytes::push_u16_be(&mut buf, token_chain);
        vector::append(&mut buf, external_address::to_bytes(recipient));
        bytes::push_u16_be(&mut buf, recipient_chain);
        bytes::push_u256_be(&mut buf, normalized_amount::to_u256(relayer_fee));

        buf
    }

    #[test_only]
    public fun amount(self: &Transfer): NormalizedAmount {
        self.amount
    }

    #[test_only]
    public fun raw_amount(self: &Transfer, decimals: u8): u64 {
        normalized_amount::to_raw(self.amount, decimals)
    }

    #[test_only]
    public fun token_address(self: &Transfer): ExternalAddress {
        self.token_address
    }

    #[test_only]
    public fun token_chain(self: &Transfer): u16 {
        self.token_chain
    }

    #[test_only]
    public fun recipient(self: &Transfer): ExternalAddress {
        self.recipient
    }

    #[test_only]
    public fun recipient_as_address(self: &Transfer): address {
        external_address::to_address(self.recipient)
    }

    #[test_only]
    public fun recipient_chain(self: &Transfer): u16 {
        self.recipient_chain
    }

    #[test_only]
    public fun relayer_fee(self: &Transfer): NormalizedAmount {
        self.relayer_fee
    }

    #[test_only]
    public fun raw_relayer_fee(self: &Transfer, decimals: u8): u64 {
        normalized_amount::to_raw(self.relayer_fee, decimals)
    }

    #[test_only]
    public fun destroy(transfer: Transfer) {
        unpack(transfer);
    }
}

#[test_only]
module token_bridge::transfer_tests {
    use std::vector::{Self};
    use wormhole::external_address::{Self};

    use token_bridge::dummy_message::{Self};
    use token_bridge::transfer::{Self};
    use token_bridge::normalized_amount::{Self};

    #[test]
    public fun test_serialize_deserialize() {
        let decimals = 8;
        let expected_amount = normalized_amount::from_raw(234567890, decimals);
        let expected_token_address = external_address::from_any_bytes(x"beef");
        let expected_token_chain = 1;
        let expected_recipient = external_address::from_any_bytes(x"cafe");
        let expected_recipient_chain = 7;
        let expected_relayer_fee =
            normalized_amount::from_raw(123456789, decimals);

        let serialized =
            transfer::serialize(
                transfer::new(
                    expected_amount,
                    expected_token_address,
                    expected_token_chain,
                    expected_recipient,
                    expected_recipient_chain,
                    expected_relayer_fee,
                )
            );
        assert!(serialized == dummy_message::encoded_transfer(), 0);

        let (
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            relayer_fee
        ) = transfer::unpack(transfer::deserialize(serialized));
        assert!(amount == expected_amount, 0);
        assert!(token_address == expected_token_address, 0);
        assert!(token_chain == expected_token_chain, 0);
        assert!(recipient == expected_recipient, 0);
        assert!(recipient_chain == expected_recipient_chain, 0);
        assert!(relayer_fee == expected_relayer_fee, 0);
    }

    #[test]
    #[expected_failure(abort_code = transfer::E_INVALID_ACTION)]
    public fun test_cannot_deserialize_invalid_payload() {
        let invalid_payload = dummy_message::encoded_transfer_with_payload();

        // Show that the first byte is not the expected payload ID.
        assert!(
            *vector::borrow(&invalid_payload, 0) != transfer::payload_id(),
            0
        );

        // You shall not pass!
        let parsed = transfer::deserialize(invalid_payload);

        // Clean up.
        transfer::destroy(parsed);
    }
}
