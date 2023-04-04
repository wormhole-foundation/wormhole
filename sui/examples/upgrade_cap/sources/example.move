/// A simple contracts that demonstrates how to send messages with wormhole.
module upgrade_cap::example {
    use sui::event::{Self};
    use sui::object::{Self, UID};
    use sui::package::{UpgradeCap};
    use sui::transfer::{Self};
    use sui::tx_context::{TxContext};

    struct State has key, store {
        id: UID,
        upgrade_cap: UpgradeCap,
    }

    struct Message has store, copy, drop {
        payload: vector<u8>,
    }

    /// Register ourselves as a wormhole emitter. This gives back an
    /// `EmitterCap` which will be required to send messages through
    /// wormhole.
    public entry fun init_with_params(
        upgrade_cap: UpgradeCap,
        ctx: &mut TxContext
    ) {
        let state = State {
            id: object::new(ctx),
            upgrade_cap: upgrade_cap,
        };
        transfer::share_object(state);
    }

    public entry fun send_message_entry(
        _state: &mut State,
        _ctx: &mut TxContext
    ) {
        event::emit(
            Message {
                payload: b"Hello world!",
            }
        );
    }
}
