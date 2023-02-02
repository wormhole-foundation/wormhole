module wormhole::wormhole {
    use sui::sui::{SUI};
    use sui::coin::{Self, Coin};
    use sui::tx_context::{Self, TxContext};
    use sui::transfer::{Self};

    //use wormhole::structs::{create_guardian, create_guardian_set};
    use wormhole::state::{Self, State};
    use wormhole::emitter::{Self};

    const E_INSUFFICIENT_FEE: u64 = 0;
    const E_TOO_MUCH_FEE: u64 = 1;

// -----------------------------------------------------------------------------
// Sending messages
    public fun publish_message(
        emitter_cap: &mut emitter::EmitterCapability,
        state: &mut State,
        nonce: u64,
        payload: vector<u8>,
        message_fee: Coin<SUI>,
    ): u64 {
        // ensure that provided fee is sufficient to cover message fees
        let expected_fee = state::get_message_fee(state);
        let val = coin::value(&message_fee);
        if (expected_fee != val){
            // if fee amount is not exactly equal to expected_fee, then throw one of two errors
            assert!(expected_fee < coin::value(&message_fee), E_TOO_MUCH_FEE);
            assert!(expected_fee > coin::value(&message_fee), E_INSUFFICIENT_FEE);
        };

        // deposit the fees into wormhole
        state::deposit_fee_coins<SUI>(state, message_fee);

        // get sequence number
        let sequence = emitter::use_sequence(emitter_cap);

        // emit event
        state::publish_event(
            emitter::get_emitter(emitter_cap),
            sequence,
            nonce,
            payload,
        );
        return sequence
    }

    public entry fun publish_message_entry(
        emitter_cap: &mut emitter::EmitterCapability,
        state: &mut State,
        nonce: u64,
        payload: vector<u8>,
        message_fee: Coin<SUI>,
    ) {
        publish_message(emitter_cap, state, nonce, payload, message_fee);
    }

    public entry fun publish_message_free(
        emitter_cap: &mut emitter::EmitterCapability,
        state: &mut State,
        nonce: u64,
        payload: vector<u8>,
    ) {
        // ensure that provided fee is sufficient to cover message fees
        let expected_fee = state::get_message_fee(state);
        assert!(expected_fee == 0, E_INSUFFICIENT_FEE);

        // get sender and sequence number
        let sequence = emitter::use_sequence(emitter_cap);

        // emit event
        state::publish_event(
            emitter::get_emitter(emitter_cap),
            sequence,
            nonce,
            payload,
        );
    }

    // -----------------------------------------------------------------------------
    // Emitter registration

    public fun register_emitter(state: &mut State, ctx: &mut TxContext): emitter::EmitterCapability {
        state::new_emitter(state, ctx)
    }

    // -----------------------------------------------------------------------------
    // get_new_emitter
    //
    // Honestly, unsure if this should survive once we get into code review but it
    // sure makes writing my test script work quite well
    //
    // This creates a new emitter object and stores it away into the senders context.
    //
    // You can then use this to call publish_message_free and generate a vaa

    public entry fun get_new_emitter(state: &mut State, ctx: &mut TxContext) {
        transfer::transfer(state::new_emitter(state, ctx), tx_context::sender(ctx));
    }
}
