module token_bridge::transfer_result {
    use wormhole::u256::{U256};
    use wormhole::u16::{U16};

    friend token_bridge::bridge_state;

    struct TransferResult has key, store, drop {
        // Chain ID of the token
        token_chain: U16,
        // Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: vector<u8>,
        // Amount being transferred (big-endian uint256)
        normalized_amount: U256,
        // Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
        normalized_relayer_fee: U256,
        // Portion of msg.value to be paid as the core bridge fee
        wormhole_fee: U256,
    }

    public fun get_token_chain(a: &TransferResult): U16 {
        a.token_chain
    }

    public fun get_token_address(a: &TransferResult): vector<u8> {
        a.token_address
    }

    public fun get_normalized_amount(a: &TransferResult): U256 {
        a.normalized_amount
    }

    public fun get_normalized_relayer_fee(a: &TransferResult): U256 {
        a.normalized_relayer_fee
    }

    public fun get_wormhole_fee(a: &TransferResult): U256 {
        a.wormhole_fee
    }

    public(friend) fun create(
        token_chain: U16,
        token_address: vector<u8>,
        normalized_amount: U256,
        normalized_relayer_fee: U256,
        wormhole_fee: U256,
        ): TransferResult {
            TransferResult {
                token_chain,
                token_address,
                normalized_amount,
                normalized_relayer_fee,
                wormhole_fee
            }
    }

}
