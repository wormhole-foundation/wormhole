module Wormhole::Governance {
    use Wormhole::Deserialize;
    use Wormhole::cursor::{Self};
    use Wormhole::VAA::{Self};
    use Wormhole::State::{updateGuardianSetIndex, storeGuardianSet, getCurrentGuardianSet};
    use Wormhole::Structs::{Guardian, GuardianSet, createGuardian, createGuardianSet};
    use Wormhole::u32::{U32};
    use 0x1::vector::{Self};

    const E_WRONG_GUARDIAN_LEN: u64 = 0x0;
    const E_NO_GUARDIAN_SET: u64    = 0x1;
    const E_INVALID_MODULE: u64     = 0x2;
    const E_INVALID_ACTION: u64     = 0x3;

    struct GuardianSetUpgrade has key{
        new_index: U32,
        guardians: vector<Guardian>,
    }

    public entry fun update_guardian_set(vaa: vector<u8>) {
        let vaa = VAA::parse_and_verify(vaa);

        let payload = VAA::destroy(vaa);

        // Verify Governance Update.
        let update = parse(payload);

        verify(&update, getCurrentGuardianSet());

        let GuardianSetUpgrade {
            new_index,
            guardians,
        } = update;

        updateGuardianSetIndex(new_index);
        storeGuardianSet(createGuardianSet(new_index, guardians), new_index);
        // TODO: when subtraction is implemented for U32, expire prev guardian set
        //expireGuardianSet(new_index-1);
    }

    public entry fun parse(bytes: vector<u8>): GuardianSetUpgrade {
        let cur = cursor::init(bytes);
        let guardians = vector::empty<Guardian>();

        let target_module = Deserialize::deserialize_vector(&mut cur, 32);
        let expected_module = x"00000000000000000000000000000000000000000000000000000000436f7265"; // Core
        assert!(target_module == expected_module, E_INVALID_MODULE);

        let action = Deserialize::deserialize_u8(&mut cur);
        assert!(action == 0x02, E_INVALID_ACTION);

        let new_index = Deserialize::deserialize_u32(&mut cur);
        let guardian_len = Deserialize::deserialize_u8(&mut cur);

        // TODO - the following assert when we can compare U16 types using (<, >, =)
        //assert!(guardian_len < 19, E_WRONG_GUARDIAN_LEN);

        while ({
            spec {
                invariant guardian_len >= 0;
                invariant guardian_len < 19;
            };
            guardian_len > 0
        }) {
            let key = Deserialize::deserialize_vector(&mut cur, 20);
            vector::push_back(&mut guardians, createGuardian(key));
            guardian_len = guardian_len - 1;
        };

        cursor::destroy_empty(cur);

        GuardianSetUpgrade {
            new_index:          new_index,
            guardians:          guardians,
        }
    }

    public entry fun verify(_update: &GuardianSetUpgrade, _previous: GuardianSet) {
        //TODO: compare indices once comparison operator is implemented for U32
        //assert!(update.new_index > getGuardianSetIndex(previous), 0);
    }
}
