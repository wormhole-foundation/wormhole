module Wormhole::Governance {
    use Wormhole::Deserialize;
    use Wormhole::cursor::{Self};
    use Wormhole::VAA::{Self};
    use Wormhole::State::{updateGuardianSetIndex, storeGuardianSet, getCurrentGuardianSet};
    use Wormhole::Structs::{Guardian, GuardianSet, createGuardian, createGuardianSet};
    use Wormhole::u32::{U32};
    use Wormhole::u16;
    use 0x1::vector::{Self};

    const E_WRONG_GUARDIAN_LEN: u64 = 0x0;
    const E_NO_GUARDIAN_SET: u64    = 0x1;
    const E_INVALID_MODULE: u64     = 0x2;
    const E_INVALID_ACTION: u64     = 0x3;
    const E_INVALID_TARGET: u64     = 0x4;

    struct GuardianSetUpgrade has key {
        new_index: U32,
        guardians: vector<Guardian>,
    }

    public entry fun update_guardian_set(vaa: vector<u8>) {
        let vaa = VAA::parse_and_verify(vaa);

        let payload = VAA::destroy(vaa);

        // Verify Governance Update.
        let update = parse_guardian_set_upgrade(payload);

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

    public entry fun parse_guardian_set_upgrade(bytes: vector<u8>): GuardianSetUpgrade {
        let cur = cursor::init(bytes);
        let guardians = vector::empty<Guardian>();

        let target_module = Deserialize::deserialize_vector(&mut cur, 32);
        let expected_module = x"00000000000000000000000000000000000000000000000000000000436f7265"; // Core
        assert!(target_module == expected_module, E_INVALID_MODULE);

        let action = Deserialize::deserialize_u8(&mut cur);
        assert!(action == 0x02, E_INVALID_ACTION);

        let chain = Deserialize::deserialize_u16(&mut cur);
        assert!(chain == u16::from_u64(0x00), E_INVALID_TARGET);

        let new_index = Deserialize::deserialize_u32(&mut cur);
        let guardian_len = Deserialize::deserialize_u8(&mut cur);

        while (guardian_len > 0) {
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

    #[test]
    public fun test_parse_guardian_set_upgrade() {
        use Wormhole::u32;
        use Wormhole::Structs::{createGuardian};

        let b = x"00000000000000000000000000000000000000000000000000000000436f7265020000000000011358cc3ae5c097b213ce3c81979e1b9f9570746aa5ff6cb952589bde862c25ef4392132fb9d4a42157114de8460193bdf3a2fcf81f86a09765f4762fd1107a0086b32d7a0977926a205131d8731d39cbeb8c82b2fd82faed2711d59af0f2499d16e726f6b211b39756c042441be6d8650b69b54ebe715e234354ce5b4d348fb74b958e8966e2ec3dbd4958a7cdeb5f7389fa26941519f0863349c223b73a6ddee774a3bf913953d695260d88bc1aa25a4eee363ef0000ac0076727b35fbea2dac28fee5ccb0fea768eaf45ced136b9d9e24903464ae889f5c8a723fc14f93124b7c738843cbb89e864c862c38cddcccf95d2cc37a4dc036a8d232b48f62cdd4731412f4890da798f6896a3331f64b48c12d1d57fd9cbe7081171aa1be1d36cafe3867910f99c09e347899c19c38192b6e7387ccd768277c17dab1b7a5027c0b3cf178e21ad2e77ae06711549cfbb1f9c7a9d8096e85e1487f35515d02a92753504a8d75471b9f49edb6fbebc898f403e4773e95feb15e80c9a99c8348d";
        let GuardianSetUpgrade { new_index, guardians } = parse_guardian_set_upgrade(b);
        assert!(new_index == u32::from_u64(1), 0);
        assert!(vector::length(&guardians) == 19, 0);
        let expected = vector::empty();
        vector::push_back(&mut expected, createGuardian(x"58cc3ae5c097b213ce3c81979e1b9f9570746aa5"));
        vector::push_back(&mut expected, createGuardian(x"ff6cb952589bde862c25ef4392132fb9d4a42157"));
        vector::push_back(&mut expected, createGuardian(x"114de8460193bdf3a2fcf81f86a09765f4762fd1"));
        vector::push_back(&mut expected, createGuardian(x"107a0086b32d7a0977926a205131d8731d39cbeb"));
        vector::push_back(&mut expected, createGuardian(x"8c82b2fd82faed2711d59af0f2499d16e726f6b2"));
        vector::push_back(&mut expected, createGuardian(x"11b39756c042441be6d8650b69b54ebe715e2343"));
        vector::push_back(&mut expected, createGuardian(x"54ce5b4d348fb74b958e8966e2ec3dbd4958a7cd"));
        vector::push_back(&mut expected, createGuardian(x"eb5f7389fa26941519f0863349c223b73a6ddee7"));
        vector::push_back(&mut expected, createGuardian(x"74a3bf913953d695260d88bc1aa25a4eee363ef0"));
        vector::push_back(&mut expected, createGuardian(x"000ac0076727b35fbea2dac28fee5ccb0fea768e"));
        vector::push_back(&mut expected, createGuardian(x"af45ced136b9d9e24903464ae889f5c8a723fc14"));
        vector::push_back(&mut expected, createGuardian(x"f93124b7c738843cbb89e864c862c38cddcccf95"));
        vector::push_back(&mut expected, createGuardian(x"d2cc37a4dc036a8d232b48f62cdd4731412f4890"));
        vector::push_back(&mut expected, createGuardian(x"da798f6896a3331f64b48c12d1d57fd9cbe70811"));
        vector::push_back(&mut expected, createGuardian(x"71aa1be1d36cafe3867910f99c09e347899c19c3"));
        vector::push_back(&mut expected, createGuardian(x"8192b6e7387ccd768277c17dab1b7a5027c0b3cf"));
        vector::push_back(&mut expected, createGuardian(x"178e21ad2e77ae06711549cfbb1f9c7a9d8096e8"));
        vector::push_back(&mut expected, createGuardian(x"5e1487f35515d02a92753504a8d75471b9f49edb"));
        vector::push_back(&mut expected, createGuardian(x"6fbebc898f403e4773e95feb15e80c9a99c8348d"));

        assert!(expected == guardians, 0);
    }

    public entry fun verify(_update: &GuardianSetUpgrade, _previous: GuardianSet) {
        //TODO: compare indices once comparison operator is implemented for U32
        //assert!(update.new_index > getGuardianSetIndex(previous), 0);
    }
}
