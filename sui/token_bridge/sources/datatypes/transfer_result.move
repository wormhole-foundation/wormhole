module token_bridge::transfer_result {
    use wormhole::external_address::ExternalAddress;

    use token_bridge::normalized_amount::NormalizedAmount;

    struct TransferResult {
        /// Chain ID of the token
        token_chain: u16,
        /// Address of the token. Left-zero-padded if shorter than 32 bytes
        token_address: ExternalAddress,
        /// Amount being transferred
        amount: NormalizedAmount,
        /// Amount of tokens that the user is willing to pay as relayer fee.
        /// Must be <= Amount.
        relayer_fee: NormalizedAmount,
    }

    public fun new(
        token_chain: u16,
        token_address: ExternalAddress,
        amount: NormalizedAmount,
        relayer_fee: NormalizedAmount,
        ): TransferResult {
            TransferResult {
                token_chain,
                token_address,
                amount,
                relayer_fee,
            }
    }

    public fun destroy(
        result: TransferResult
    ): (
        u16,
        ExternalAddress,
        NormalizedAmount,
        NormalizedAmount
    ) {
        let TransferResult {
            token_chain,
            token_address,
            amount,
            relayer_fee
        } = result;
        (token_chain, token_address, amount, relayer_fee)
    }
}
