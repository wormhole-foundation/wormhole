module Wormhole::Governance {
    use Wormhole::Deserialize;
    use Wormhole::cursor::{Self};
    use Wormhole::VAA::{Self};
    use Wormhole::State::{updateGuardianSetIndex, storeGuardianSet, expireGuardianSet, getCurrentGuardianSet};
    use Wormhole::Structs::{Guardian, GuardianSet, createGuardian, createGuardianSet, getGuardianSetIndex};
    use Wormhole::Uints::{U32};
    use 0x1::vector::{Self};
    use 0x1::string::{Self, String};

    const E_WRONG_GUARDIAN_LEN: u64 = 0x0;
    const E_NO_GUARDIAN_SET: u64    = 0x1;

    struct GuardianUpdate has key{
        guardian_module:    vector<u8>,
        action:             u8,
        new_index:          U32,
        guardians:          vector<Guardian>,
    }

    public entry fun update_guardian_set(vaa: vector<u8>): (bool, String) {
        let (vaa, valid, reason) = VAA::parseAndVerifyVAA(vaa);

        let payload = VAA::destroy(vaa);

        if (!valid) {
            return (false, reason)
        };

        // Verify Governance Update.
        let update = parse(payload);

        verify(&update, getCurrentGuardianSet());

        let GuardianUpdate {
            guardian_module: _,
            action: _, //action
            new_index,
            guardians,
        } = update;

        updateGuardianSetIndex(new_index);
        storeGuardianSet(createGuardianSet(new_index, guardians), new_index);
        // TODO: when subtraction is implemented for U32, expire prev guardian set
        //expireGuardianSet(new_index-1);
        return (true, string::utf8(b""))
    }

    public entry fun parse(bytes: vector<u8>): GuardianUpdate {
        let cur = cursor::init(bytes);
        let guardians = vector::empty<Guardian>();
        let guardian_module = Deserialize::deserialize_vector(&mut cur, 32);
        //TODO: missing chainID?
        let action  = Deserialize::deserialize_u8(&mut cur);
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

        GuardianUpdate {
            guardian_module:    guardian_module,
            action:             action,
            new_index:          new_index,
            guardians:          guardians,
        }
    }

    public entry fun verify(update: &GuardianUpdate, previous: GuardianSet){
        let (guardian_module, action) = (update.guardian_module, update.action);
        assert!(vector::length(&guardian_module) == 32, 0);
        assert!(action == 0x02, 0);

        //TODO: compare indices once comparison operator is implemented for U32
        //assert!(update.new_index > getGuardianSetIndex(previous), 0);
    }
}
