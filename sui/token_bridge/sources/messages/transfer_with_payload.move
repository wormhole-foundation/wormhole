module token_bridge::transfer_with_payload {
    use std::vector;
    use wormhole::bytes::{
        push_u8,
        push_u16_be,
    };
    use wormhole::bytes::{
        take_u8,
        take_u16_be,
    };
    use wormhole::cursor;

    use wormhole::external_address::{Self, ExternalAddress};

    use token_bridge::normalized_amount::{Self, NormalizedAmount};

    const E_INVALID_ACTION: u64 = 0;

    const PAYLOAD_ID: u8 = 3;

    struct TransferWithPayload has store, drop {
        // Amount being transferred (big-endian uint256).
        amount: NormalizedAmount,
        // Address of the token. Left-zero-padded if shorter than 32 bytes.
        token_address: ExternalAddress,
        // Chain ID of the token.
        token_chain: u16,
        // Address of the recipient. Left-zero-padded if shorter than 32 bytes.
        recipient: ExternalAddress,
        // Chain ID of the recipient.
        recipient_chain: u16,
        // Address of the message sender. Left-zero-padded if shorter than 32 bytes.
        sender: ExternalAddress,
        // An arbitrary payload.
        payload: vector<u8>,
    }

    public fun new(
        amount: NormalizedAmount,
        token_address: ExternalAddress,
        token_chain: u16,
        recipient: ExternalAddress,
        recipient_chain: u16,
        sender: ExternalAddress,
        payload: vector<u8>
    ): TransferWithPayload {
        TransferWithPayload {
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            sender,
            payload,
        }
    }

    public fun amount(self: &TransferWithPayload): NormalizedAmount {
        self.amount
    }

    public fun token_address(self: &TransferWithPayload): ExternalAddress {
        self.token_address
    }

    public fun token_chain(self: &TransferWithPayload): u16 {
        self.token_chain
    }

    public fun recipient(self: &TransferWithPayload): ExternalAddress {
        self.recipient
    }

    public fun recipient_chain(self: &TransferWithPayload): u16 {
        self.recipient_chain
    }

    public fun sender(self: &TransferWithPayload): ExternalAddress {
        self.sender
    }

    public fun payload(self: &TransferWithPayload): vector<u8> {
        self.payload
    }

    public fun serialize(transfer: TransferWithPayload): vector<u8> {
        let encoded = vector::empty<u8>();
        push_u8(&mut encoded, PAYLOAD_ID);
        normalized_amount::serialize_be(&mut encoded, transfer.amount);
        vector::append(
            &mut encoded,
            external_address::to_bytes(transfer.token_address)
        );
        push_u16_be(&mut encoded, transfer.token_chain);
        vector::append(
            &mut encoded,
            external_address::to_bytes(transfer.recipient)
        );
        push_u16_be(&mut encoded, transfer.recipient_chain);
        vector::append(
            &mut encoded,
            external_address::to_bytes(transfer.sender),
        );
        vector::append(&mut encoded, transfer.payload);
        encoded
    }

    public fun deserialize(transfer: vector<u8>): TransferWithPayload {
        let cur = cursor::new(transfer);
        assert!(take_u8(&mut cur) == PAYLOAD_ID, E_INVALID_ACTION);
        let amount = normalized_amount::deserialize_be(&mut cur);
        let token_address = external_address::take_bytes(&mut cur);
        let token_chain = take_u16_be(&mut cur);
        let recipient = external_address::take_bytes(&mut cur);
        let recipient_chain = take_u16_be(&mut cur);
        let sender = external_address::take_bytes(&mut cur);
        let payload = cursor::take_rest(cur);
        new(
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            sender,
            payload
        )
    }
}

#[test_only]
module token_bridge::transfer_with_payload_test {
    use wormhole::external_address::{Self};

    use token_bridge::transfer_with_payload::{Self};
    use token_bridge::normalized_amount::{Self};

    #[test]
    fun test_transfer_with_payload(){
        let amount = normalized_amount::default();
        let token_address = external_address::from_any_bytes(x"0011223344");
        let recipient = external_address::from_any_bytes(x"003456");
        let sender = external_address::from_any_bytes(x"99887766");
        let payload = x"12334435345345234234";

        let transfer_with_payload = transfer_with_payload::new(
            amount, // amount
            token_address, // token address
            3, // token chain
            recipient, // recipient
            6, // recipient chain
            sender, // sender
            payload // payload
        );
        // Serialize and deserialize TransferWithPayload object.
        let se = transfer_with_payload::serialize(transfer_with_payload);
        let de = transfer_with_payload::deserialize(se);

        // Test that the object fields are unchanged.
        assert!(transfer_with_payload::amount(&de) == amount, 0);
        assert!(transfer_with_payload::token_address(&de) == token_address, 0);
        assert!(transfer_with_payload::token_chain(&de) == 3, 0);
        assert!(transfer_with_payload::recipient(&de) == recipient, 0);
        assert!(transfer_with_payload::recipient_chain(&de) == 6, 0);
        assert!(transfer_with_payload::sender(&de) == sender, 0);
        assert!(transfer_with_payload::payload(&de) == payload, 0);
    }
}
