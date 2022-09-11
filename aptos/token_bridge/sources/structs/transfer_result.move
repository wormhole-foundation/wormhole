module token_bridge::transfer_result {
    use wormhole::u256::{U256};
    use wormhole::u16::{U16};
    use wormhole::external_address::{Self, ExternalAddress};

    friend token_bridge::transfer_tokens;

    struct TransferResult {
        // Chain ID of the token
        token_chain: U16,
        // Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: ExternalAddress,
        // Amount being transferred (big-endian uint256)
        normalized_amount: U256,
        // Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
        normalized_relayer_fee: U256,
    }

    public fun destroy(a: TransferResult): (U16, ExternalAddress, U256, U256) {
        let TransferResult {
            token_chain,
            token_address,
            normalized_amount,
            normalized_relayer_fee
        } = a;
        (token_chain, token_address, normalized_amount, normalized_relayer_fee)
    }

    public(friend) fun create(
        token_chain: U16,
        token_address: vector<u8>,
        normalized_amount: U256,
        normalized_relayer_fee: U256,
        ): TransferResult {
            TransferResult {
                token_chain,
                token_address: external_address::from_vector(token_address),
                normalized_amount,
                normalized_relayer_fee,
            }
    }

}
