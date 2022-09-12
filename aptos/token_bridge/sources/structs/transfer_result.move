module token_bridge::transfer_result {
    use wormhole::u16::U16;
    use wormhole::external_address::ExternalAddress;

    use token_bridge::normalized_amount::NormalizedAmount;

    friend token_bridge::transfer_tokens;

    struct TransferResult {
        /// Chain ID of the token
        token_chain: U16,
        /// Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: ExternalAddress,
        /// Amount being transferred
        normalized_amount: NormalizedAmount,
        /// Amount of tokens that the user is willing to pay as relayer fee. Must be <= Amount.
        normalized_relayer_fee: NormalizedAmount,
    }

    public fun destroy(a: TransferResult): (U16, ExternalAddress, NormalizedAmount, NormalizedAmount) {
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
        token_address: ExternalAddress,
        normalized_amount: NormalizedAmount,
        normalized_relayer_fee: NormalizedAmount,
        ): TransferResult {
            TransferResult {
                token_chain,
                token_address,
                normalized_amount,
                normalized_relayer_fee,
            }
    }

}
