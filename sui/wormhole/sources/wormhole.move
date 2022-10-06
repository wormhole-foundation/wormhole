module wormhole::wormhole {
    use sui::sui::{SUI};
    use sui::coin::{Self, Coin};
    use sui::tx_context::{Self, TxContext};
    use sui::transfer::{Self};

    //use wormhole::structs::{create_guardian, create_guardian_set};
    use wormhole::state::{Self, State};

    // use wormhole::myu16 as u16;
    // use wormhole::myu32::{Self as u32, U32};
    // use wormhole::external_address::{Self};

    const E_INSUFFICIENT_FEE: u64 = 0;

// -----------------------------------------------------------------------------
// Sending messages
    public entry fun publish_message(
        state: &mut State,
        nonce: u64,
        payload: vector<u8>,
        message_fee: Coin<SUI>,
        ctx: &mut TxContext,
    ) {
        // ensure that provided fee is sufficient to cover message fees
        let expected_fee = state::get_message_fee(state);
        assert!(expected_fee <= coin::value(&message_fee), E_INSUFFICIENT_FEE);
        // deposit the fees into the wormhole account
        transfer::transfer(message_fee, @wormhole);
        let sender = &tx_context::sender(ctx);
        let sequence = state::get_sequence(state, sender);
        state::publish_event(
            sequence,
            nonce,
            payload,
            ctx,
        );
        state::increase_sequence(state, sender);
        //sequence
    }
}
