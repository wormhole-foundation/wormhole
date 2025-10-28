    use sui::tx_context::TxContext;
    // Highly unsafe token transfer method, that accepts an `amount_to_bridge`
    // that is actually passed to the message publication.
    public fun prepare_transfer_unsafe<CoinType>(
        asset_info: VerifiedAsset<CoinType>,
        funded: Coin<CoinType>,
        amount_to_bridge: u64,
        recipient_chain: u16,
        recipient: vector<u8>,
        relayer_fee: u64,
        nonce: u32
    ): (
        TransferTicket<CoinType>,
        Coin<CoinType>
    ) {
        let (
            bridged_in,
            _
        ) = take_truncated_amount(&asset_info, &mut funded);

        let decimals = token_registry::coin_decimals(&asset_info);
        let norm_amount = normalized_amount::from_raw(amount_to_bridge, decimals);

        let ticket =
            TransferTicket {
                asset_info,
                bridged_in,
                norm_amount,
                relayer_fee,
                recipient_chain,
                recipient,
                nonce
            };

        // The remaining amount of funded may have dust depending on the
        // decimals of this asset.
        (ticket, funded)
    }

    public fun transfer_tokens_unsafe<CoinType>(
        token_bridge_state: &mut State,
        ticket: TransferTicket<CoinType>,
        ctx: &mut TxContext
    ): (
        MessageTicket, 
        Coin<CoinType>
    ) {
        // This capability ensures that the current build version is used.
        let latest_only = state::assert_latest_only(token_bridge_state);

        let TransferTicket {
            asset_info,
            bridged_in,
            norm_amount,
            recipient_chain,
            recipient,
            relayer_fee,
            nonce
        } = ticket;

        // Ensure that the recipient is a 32-byte address.
        let recipient = external_address::new(bytes32::from_bytes(recipient));

        let token_chain = token_registry::token_chain(&asset_info);
        let token_address = token_registry::token_address(&asset_info);

        let encoded_transfer =
            transfer::serialize(
                transfer::new(
                    norm_amount,
                    token_address,
                    token_chain,
                    recipient,
                    recipient_chain,
                    normalized_amount::from_raw(
                        relayer_fee,
                        token_registry::coin_decimals(&asset_info)
                    )
                )
            );

        // Prepare Wormhole message with encoded `Transfer`.
        (state::prepare_wormhole_message(
            &latest_only,
            token_bridge_state,
            nonce,
            encoded_transfer
        ), coin::from_balance(bridged_in, ctx))
    }