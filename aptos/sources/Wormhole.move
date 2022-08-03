module Wormhole::Wormhole {
    use Wormhole::Governance;
    use Wormhole::State::{WormholeMessage, GuardianSetChanged};
    use 0x1::event::{Self, EventHandle};
    use 0x1::signer::{address_of};

    struct WormholeMessageHandle has key {
        event: EventHandle<WormholeMessage>
    }

     struct GuardianSetChangedHandle has key {
        event: EventHandle<GuardianSetChanged>
    }

    fun init(admin: &signer) {
        Governance::init_guardian_set(admin);
        move_to(admin, WormholeMessageHandle{
            event: event::new_event_handle<WormholeMessage>(admin)
        });
         move_to(admin, GuardianSetChangedHandle{
            event: event::new_event_handle<GuardianSetChanged>(admin)
        });

    }
}

