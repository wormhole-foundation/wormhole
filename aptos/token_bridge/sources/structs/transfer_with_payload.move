module token_bridge::transfer_with_payload {
    use 0x1::vector::{Self};
    use wormhole::serialize::{serialize_u8, serialize_u16, serialize_u256, serialize_vector};
    use wormhole::deserialize::{deserialize_u8, deserialize_u16, deserialize_u256, deserialize_vector};
    use wormhole::cursor::{Self};

    use wormhole::u256::{U256};
    use wormhole::u16::{U16};

    friend token_bridge::transfer_tokens;

    struct TransferWithPayload has key, store, drop {
        // PayloadID uint8 = 3
        payload_id: u8,
        // Amount being transferred (big-endian uint256)
        amount: U256,
        // Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: vector<u8>,
        // Chain ID of the token
        token_chain: U16,
        // Address of the recipient. Left-zero-padded if shorter than 32 bytes
        to: vector<u8>,
        // Chain ID of the recipient
        to_chain: U16,
        // Address of the message sender. Left-zero-padded if shorter than 32 bytes
        from_address: vector<u8>,
        // An arbitrary payload
        payload: vector<u8>,
    }

    public fun get_payload_id(a: &TransferWithPayload): u8 {
        a.payload_id
    }

    public fun get_amount(a: &TransferWithPayload): U256 {
        a.amount
    }

    public fun get_token_address(a: &TransferWithPayload): vector<u8> {
        a.token_address
    }

    public fun get_token_chain(a: &TransferWithPayload): U16 {
        a.token_chain
    }

    public fun get_to(a: &TransferWithPayload): vector<u8> {
        a.to
    }

    public fun get_to_chain(a: &TransferWithPayload): U16 {
        a.to_chain
    }

    public fun get_from_address(a: &TransferWithPayload): vector<u8> {
        a.from_address
    }

    public fun get_payload(a: &TransferWithPayload): vector<u8> {
        a.payload
    }

    public(friend) fun create(
        amount: U256,
        token_address: vector<u8>,
        token_chain: U16,
        to: vector<u8>,
        to_chain: U16,
        from_address: vector<u8>,
        payload: vector<u8>
    ): TransferWithPayload {
        TransferWithPayload {
            payload_id: 3,
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
        serialize_u8(&mut encoded, transfer.payload_id);
        serialize_u256(&mut encoded, transfer.amount);
        serialize_vector(&mut encoded, transfer.token_address);
        serialize_u16(&mut encoded, transfer.token_chain);
        serialize_vector(&mut encoded, transfer.to);
        serialize_u16(&mut encoded, transfer.to_chain);
        serialize_vector(&mut encoded, transfer.from_address);
        serialize_vector(&mut encoded, transfer.payload);
        encoded
    }

    public fun parse(transfer: vector<u8>): TransferWithPayload {
        let cur = cursor::init(transfer);
        let payload_id = deserialize_u8(&mut cur);
        let amount = deserialize_u256(&mut cur);
        let token_address = deserialize_vector(&mut cur, 32);
        let token_chain = deserialize_u16(&mut cur);
        let to = deserialize_vector(&mut cur, 32);
        let to_chain = deserialize_u16(&mut cur);
        let from_address = deserialize_vector(&mut cur, 32);
        let payload = cursor::rest(cur);
        TransferWithPayload {
            payload_id,
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
