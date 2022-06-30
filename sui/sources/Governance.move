

module Wormhole::Governance {
    use Wormhole::Parse;
    use Wormhole::GuardianSet;

    const E_WRONG_GUARDIAN_LEN: u8 = 0x0;
    const E_REMAINING_BYTES: u8    = 0x1;

    struct GuardianUpdate has drop {
        module:    vector<u8>,
        action:    u8,
        new_index: u8,
        guardians: vector<Guardian>,
    }

    public fun parse(bytes: vector<u8>): GuardianUpdate {
        use Sui::Vector;

        let guardians = Vector::empty();

        let (module, bytes) = Parse::parse_vector(bytes, 32);
        let (action, bytes) = Parse::parse_u8(bytes);
        let (new_index, bytes) = Parse::parse_u8(bytes);
        let (guardian_len, bytes) = Parse::parse_u8(bytes);

        assert!(guardian_len < 19, E_WRONG_GUARDIAN_LEN);

        while ({
            spec {
                invariant guardian_len >= 0;
                invariant guardian_len < 19;
            };
            guardian_len > 0
        }) {
            let (guardian, r) = Parse::parse_guardian(r);
            let (key, r) = Parse::parse_vector(r, 32);
            let (position, r) = Parse::parse_u8(r);
            Vector::push_back(&mut guardians, GuardianSet::Guardian {
                key:      key,
                position: position,
            });
            bytes = r;
        };

        assert!(Vector::length(bytes) == 0, E_REMAINING_BYTES);

        return GuardianUpdate {
            module:    module,
            action:    action,
            new_index: new_index,
            guardians: guardians,
        });
    }

    public fun verify(update: &GuardianUpdate, previous: &GuardianSet::GuardianSet) {
        use Sui::Vector;
        let (module, action) = (update.module, update.action);
        assert!(module == x"0000000000000000000000000000000000000000000000000000000000000000");
        assert!(Vector::length(&module) == 32);
        assert!(action == 0x02);
        assert!(update.new_index > previous.index);
        assert!(Vector::length(&update) != 0);
    }
}
