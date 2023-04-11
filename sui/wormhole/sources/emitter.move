// SPDX-License-Identifier: Apache 2

/// This module implements a capability (`EmitterCap`), which allows one to send
/// Wormhole messages. Its external address is determined by the capability's
/// `id`, which is a 32-byte vector.
module wormhole::emitter {
    use sui::event::{Self};
    use sui::object::{Self, ID, UID};
    use sui::tx_context::{TxContext};

    use wormhole::state::{Self, State};
    use wormhole::version_control::{Emitter as EmitterControl};

    friend wormhole::publish_message;

    struct EmitterCreated has drop, copy {
        emitter_cap: ID
    }

    struct EmitterDestroyed has drop, copy {
        emitter_cap: ID
    }

    /// `EmitterCap` is a Sui object that gives a user or smart contract the
    /// capability to send Wormhole messages. For every Wormhole message
    /// emitted, a unique `sequence` is used.
    struct EmitterCap has key, store {
        id: UID,

        /// Sequence number of the next wormhole message.
        sequence: u64
    }

    /// Generate a new `EmitterCap`.
    public fun new(wormhole_state: &State, ctx: &mut TxContext): EmitterCap {
        state::check_minimum_requirement<EmitterControl>(wormhole_state);

        let cap =
            EmitterCap {
                id: object::new(ctx),
                sequence: 0
            };

        event::emit(EmitterCreated { emitter_cap: object::id(&cap)});

        cap
    }

    /// Returns current sequence (which will be used in the next Wormhole
    /// message emitted).
    public fun sequence(self: &EmitterCap): u64 {
        self.sequence
    }

    /// Once a Wormhole message is emitted, an `EmitterCap` upticks its
    /// internal `sequence` for the next message.
    public(friend) fun use_sequence(self: &mut EmitterCap): u64 {
        let sequence = self.sequence;
        self.sequence = sequence + 1;
        sequence
    }

    /// Destroys an `EmitterCap`.
    ///
    /// Note that this operation removes the ability to send messages using the
    /// emitter id, and is irreversible.
    public fun destroy(cap: EmitterCap) {
        event::emit(EmitterDestroyed { emitter_cap: object::id(&cap)});

        let EmitterCap { id, sequence: _ } = cap;
        object::delete(id);
    }

    #[test_only]
    public fun dummy(): EmitterCap {
        EmitterCap {
            id: object::new(&mut sui::tx_context::dummy()),
            sequence: 0
        }
    }
}

#[test_only]
module wormhole::emitter_tests {
    use sui::object::{Self};
    use sui::test_scenario::{Self};

    use wormhole::emitter::{Self};
    use wormhole::wormhole_scenario::{
        person,
        return_state,
        set_up_wormhole,
        take_state
    };

    #[test]
    public fun test_emitter() {
        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Ignore effects.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);

        let dummy_cap = emitter::dummy();
        let expected =
            @0x381dd9078c322a4663c392761a0211b527c127b29583851217f948d62131f409;
        assert!(object::id_to_address(&object::id(&dummy_cap)) == expected, 0);

        // Generate new emitter.
        let cap = emitter::new(&worm_state, test_scenario::ctx(scenario));

        // And check emitter cap's address.
        let expected =
            @0x75c3360eb19fd2c20fbba5e2da8cf1a39cdb1ee913af3802ba330b852e459e05;
        assert!(object::id_to_address(&object::id(&cap)) == expected, 0);

        // Clean up.
        emitter::destroy(dummy_cap);
        emitter::destroy(cap);
        return_state(worm_state);

        // Done.
        test_scenario::end(my_scenario);
    }
}
