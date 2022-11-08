module wormhole::wormhole {
    use sui::sui::{SUI};
    use sui::coin::{Self, Coin};
    use sui::tx_context::{TxContext};
    use sui::transfer::{Self};

    //use wormhole::structs::{create_guardian, create_guardian_set};
    use wormhole::state::{Self, State};
    use wormhole::emitter::{Self};

    // use wormhole::myu16 as u16;
    // use wormhole::myu32::{Self as u32, U32};
    // use wormhole::external_address::{Self};

    const E_INSUFFICIENT_FEE: u64 = 0;

// -----------------------------------------------------------------------------
// Sending messages
// TODO - make this a non-entry fun, so we can return the sequence number?
//        As long as it is entry, we cannot have a return value. Is it true that
//        we don't need this function to be entry, because most of the time it
//        is called by a smart contract?
    public entry fun publish_message(
        emitter_cap: &mut emitter::EmitterCapability,
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

        // get sequence number
        let sequence = emitter::use_sequence(emitter_cap);

        // emit event
        state::publish_event(
            sequence,
            nonce,
            payload,
            ctx,
        );
    }

    public entry fun publish_message_free(
        emitter_cap: &mut emitter::EmitterCapability,
        state: &mut State,
        nonce: u64,
        payload: vector<u8>,
        ctx: &mut TxContext,
    ) {
        // ensure that provided fee is sufficient to cover message fees
        let expected_fee = state::get_message_fee(state);
        assert!(expected_fee == 0, E_INSUFFICIENT_FEE);

        // get sender and sequence number
        let sequence = emitter::use_sequence(emitter_cap);

        // emit event
        state::publish_event(
            sequence,
            nonce,
            payload,
            ctx,
        );

    }

    // -----------------------------------------------------------------------------
    // Emitter registration

    public fun register_emitter(state: &mut State, ctx: &mut TxContext): emitter::EmitterCapability {
        state::new_emitter(state, ctx)
    }

}
