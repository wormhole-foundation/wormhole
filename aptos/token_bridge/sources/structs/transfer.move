module token_bridge::transfer {
    use 0x1::vector::{Self};
    use wormhole::serialize::{
        serialize_u8,
        serialize_u16,
        serialize_u256,
        serialize_vector
    };
    use wormhole::deserialize::{
        deserialize_u8,
        deserialize_u16,
        deserialize_u256,
        deserialize_vector
    };
    use wormhole::cursor::{Self};

    use wormhole::u256::{U256};
    use wormhole::u16::{U16};

    friend token_bridge::bridge_state;

    // TODO: change amount to u64, and error in the parser if more comes
    // through?

    struct Transfer has key, store, drop {
        // TODO: is there a need to store the payload id in the parsed type?
        // It's there in the wire format to instruct which type to deserialise
        // into, but here the type implicitly encodes that information already.
        // PayloadID uint8 = 1
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
        // Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
        fee: U256, //should be u256
    }

    public fun get_payload_id(a: &Transfer): u8 {
        a.payload_id
    }

    public fun get_amount(a: &Transfer): U256 {
        a.amount
    }

    public fun get_token_address(a: &Transfer): vector<u8> {
        a.token_address
    }

    public fun get_token_chain(a: &Transfer): U16 {
        a.token_chain
    }

    public fun get_to(a: &Transfer): vector<u8> {
        a.to
    }

    public fun get_to_chain(a: &Transfer): U16 {
        a.to_chain
    }

    public fun get_fee(a: &Transfer): U256 {
        a.fee
    }

    public(friend) fun create(
        payload_id: u8,
        amount: U256,
        token_address: vector<u8>,
        token_chain: U16,
        to: vector<u8>,
        to_chain: U16,
        fee: U256,
    ): Transfer {
        Transfer {
            payload_id,
            amount,
            token_address,
            token_chain,
            to,
            to_chain,
            fee,
        }
    }

    public fun parse(transfer: vector<u8>): Transfer {
        let cur = cursor::init(transfer);
        let payload_id = deserialize_u8(&mut cur);
        let amount = deserialize_u256(&mut cur);
        let token_address = deserialize_vector(&mut cur, 32);
        let token_chain = deserialize_u16(&mut cur);
        let to = deserialize_vector(&mut cur, 32);
        let to_chain= deserialize_u16(&mut cur);
        let fee = deserialize_u256(&mut cur);
        cursor::destroy_empty(cur);
        Transfer {
            payload_id,
            amount,
            token_address,
            token_chain,
            to,
            to_chain,
            fee
        }
    }

    public fun encode(transfer: Transfer): vector<u8> {
        let encoded = vector::empty<u8>();
        serialize_u8(&mut encoded, transfer.payload_id);
        serialize_u256(&mut encoded, transfer.amount);
        serialize_vector(&mut encoded, transfer.token_address);
        serialize_u16(&mut encoded, transfer.token_chain);
        serialize_vector(&mut encoded, transfer.to);
        serialize_u16(&mut encoded, transfer.to_chain);
        serialize_u256(&mut encoded, transfer.fee);
        encoded
    }

}
