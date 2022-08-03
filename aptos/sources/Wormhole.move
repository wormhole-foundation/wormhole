module Wormhole::Wormhole {
    use Wormhole::Governance::{init_guardian_set};
    use Wormhole::State::{WormholeMessage, GuardianSetChanged};
    use Wormhole::State::{initMesssageHandles};

    use 0x1::event::{Self, EventHandle};
    use 0x1::signer::{address_of};

    fun init(admin: &signer) {
        init_guardian_set(admin);
        initMesssageHandles(admin);
       
    }
}

