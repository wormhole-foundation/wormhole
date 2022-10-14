module token_bridge::transfer_with_payload {
    use std::vector;
    use wormhole::serialize::{
        serialize_u8,
        serialize_u16,
        serialize_vector,
    };
    use wormhole::deserialize::{
        deserialize_u8,
        deserialize_u16,
    };
    use wormhole::cursor;

    use wormhole::u16::U16;
    use wormhole::external_address::{Self, ExternalAddress};

    use token_bridge::normalized_amount::{Self, NormalizedAmount};

    friend token_bridge::transfer_tokens;

    const E_INVALID_ACTION: u64 = 0;

    struct TransferWithPayload has key, store, drop {
        /// Amount being transferred (big-endian uint256)
        amount: NormalizedAmount,
        /// Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: ExternalAddress,
        /// Chain ID of the token
        token_chain: U16,
        /// Address of the recipient. Left-zero-padded if shorter than 32 bytes
        to: ExternalAddress,
        /// Chain ID of the recipient
        to_chain: U16,
        /// Address of the message sender. Left-zero-padded if shorter than 32 bytes
        from_address: ExternalAddress,
        /// An arbitrary payload
        payload: vector<u8>,
    }

    public fun get_amount(a: &TransferWithPayload): NormalizedAmount {
        a.amount
    }

    public fun get_token_address(a: &TransferWithPayload): ExternalAddress {
        a.token_address
    }

    public fun get_token_chain(a: &TransferWithPayload): U16 {
        a.token_chain
    }

    public fun get_to(a: &TransferWithPayload): ExternalAddress {
        a.to
    }

    public fun get_to_chain(a: &TransferWithPayload): U16 {
        a.to_chain
    }

    public fun get_from_address(a: &TransferWithPayload): ExternalAddress {
        a.from_address
    }

    public fun get_payload(a: &TransferWithPayload): vector<u8> {
        a.payload
    }

    public(friend) fun create(
        amount: NormalizedAmount,
        token_address: ExternalAddress,
        token_chain: U16,
        to: ExternalAddress,
        to_chain: U16,
        from_address: ExternalAddress,
        payload: vector<u8>
    ): TransferWithPayload {
        TransferWithPayload {
            amount,
            token_address,
            token_chain,
            to,
            to_chain,
            from_address,
            payload,
        }
    }

    public fun encode(transfer: TransferWithPayload): vector<u8> {
        let encoded = vector::empty<u8>();
        serialize_u8(&mut encoded, 3);
        normalized_amount::serialize(&mut encoded, transfer.amount);
        external_address::serialize(&mut encoded, transfer.token_address);
        serialize_u16(&mut encoded, transfer.token_chain);
        external_address::serialize(&mut encoded, transfer.to);
        serialize_u16(&mut encoded, transfer.to_chain);
        external_address::serialize(&mut encoded, transfer.from_address);
        serialize_vector(&mut encoded, transfer.payload);
        encoded
    }

    public fun parse(transfer: vector<u8>): TransferWithPayload {
        let cur = cursor::init(transfer);
        let action = deserialize_u8(&mut cur);
        assert!(action == 3, E_INVALID_ACTION);
        let amount = normalized_amount::deserialize(&mut cur);
        let token_address = external_address::deserialize(&mut cur);
        let token_chain = deserialize_u16(&mut cur);
        let to = external_address::deserialize(&mut cur);
        let to_chain = deserialize_u16(&mut cur);
        let from_address = external_address::deserialize(&mut cur);
        let payload = cursor::rest(cur);
        TransferWithPayload {
            amount,
            token_address,
            token_chain,
            to,
            to_chain,
            from_address,
            payload
        }
    }
}
