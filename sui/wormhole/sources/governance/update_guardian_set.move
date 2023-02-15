module wormhole::update_guardian_set {
    use std::vector::{Self};
    use sui::tx_context::{TxContext};

    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};
    use wormhole::myvaa::{Self as vaa};
    use wormhole::state::{Self, State};
    use wormhole::structs::{
        Guardian,
        create_guardian,
        create_guardian_set
    };

    const E_WRONG_GUARDIAN_LEN: u64 = 0x0;
    const E_NO_GUARDIAN_SET: u64 = 0x1;
    const E_INVALID_MODULE: u64 = 0x2;
    const E_INVALID_ACTION: u64 = 0x3;
    const E_INVALID_TARGET: u64 = 0x4;
    const E_NON_INCREMENTAL_GUARDIAN_SETS: u64 = 0x5;

    struct UpdateGuardianSet {
        new_index: u32,
        guardians: vector<Guardian>,
    }

    public entry fun submit_vaa(
        state: &mut State,
        vaa: vector<u8>,
        ctx: &mut TxContext
    ) {
        let vaa = vaa::parse_and_verify(state, vaa, ctx);
        vaa::assert_governance(state, &vaa);
        vaa::replay_protect(state, &vaa);

        do_upgrade(state, parse_payload(vaa::destroy(vaa)), ctx)
    }

    fun do_upgrade(
        state: &mut State,
        upgrade: UpdateGuardianSet,
        ctx: &TxContext
    ) {
        let current_index = state::get_current_guardian_set_index(state);

        let UpdateGuardianSet {
            new_index,
            guardians,
        } = upgrade;

        assert!(
            new_index == current_index + 1,
            E_NON_INCREMENTAL_GUARDIAN_SETS
        );

        state::update_guardian_set_index(state, new_index);
        state::store_guardian_set(
            state,
            new_index,
            create_guardian_set(new_index, guardians)
        );
        state::expire_guardian_set(state, current_index, ctx);
    }

    #[test_only]
    public fun do_upgrade_test(
        s: &mut State,
        new_index: u32,
        guardians: vector<Guardian>,
        ctx: &mut TxContext
    ) {
        do_upgrade(s, UpdateGuardianSet { new_index, guardians }, ctx)
    }

    public fun parse_payload(bytes: vector<u8>): UpdateGuardianSet {
        let cur = cursor::new(bytes);
        let guardians = vector::empty<Guardian>();

        let target_module = bytes::to_bytes(&mut cur, 32);
        let expected_module =
            x"00000000000000000000000000000000000000000000000000000000436f7265"; // Core
        assert!(target_module == expected_module, E_INVALID_MODULE);

        let action = bytes::deserialize_u8(&mut cur);
        assert!(action == 0x02, E_INVALID_ACTION);

        let chain = bytes::deserialize_u16_be(&mut cur);
        assert!(chain == 0, E_INVALID_TARGET);

        let new_index = bytes::deserialize_u32_be(&mut cur);
        let guardian_len = bytes::deserialize_u8(&mut cur);

        while (guardian_len > 0) {
            let key = bytes::to_bytes(&mut cur, 20);
            vector::push_back(&mut guardians, create_guardian(key));
            guardian_len = guardian_len - 1;
        };

        cursor::destroy_empty(cur);

        UpdateGuardianSet {
            new_index,
            guardians
        }
    }

    #[test_only]
    public fun split(upgrade: UpdateGuardianSet): (u32, vector<Guardian>) {
        let UpdateGuardianSet { new_index, guardians } = upgrade;
        (new_index, guardians)
    }
}

#[test_only]
module wormhole::guardian_set_upgrade_test {
    use std::vector;

    use wormhole::structs::{create_guardian};
    use wormhole::update_guardian_set::{Self};

    use sui::test_scenario::{
        Self,
        Scenario,
        next_tx,
        take_shared,
        return_shared,
        ctx
    };
    use sui::tx_context::{increment_epoch_number};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    #[test]
    public fun test_parse_guardian_set_upgrade() {
        let b =
            x"00000000000000000000000000000000000000000000000000000000436f7265020000000000011358cc3ae5c097b213ce3c81979e1b9f9570746aa5ff6cb952589bde862c25ef4392132fb9d4a42157114de8460193bdf3a2fcf81f86a09765f4762fd1107a0086b32d7a0977926a205131d8731d39cbeb8c82b2fd82faed2711d59af0f2499d16e726f6b211b39756c042441be6d8650b69b54ebe715e234354ce5b4d348fb74b958e8966e2ec3dbd4958a7cdeb5f7389fa26941519f0863349c223b73a6ddee774a3bf913953d695260d88bc1aa25a4eee363ef0000ac0076727b35fbea2dac28fee5ccb0fea768eaf45ced136b9d9e24903464ae889f5c8a723fc14f93124b7c738843cbb89e864c862c38cddcccf95d2cc37a4dc036a8d232b48f62cdd4731412f4890da798f6896a3331f64b48c12d1d57fd9cbe7081171aa1be1d36cafe3867910f99c09e347899c19c38192b6e7387ccd768277c17dab1b7a5027c0b3cf178e21ad2e77ae06711549cfbb1f9c7a9d8096e85e1487f35515d02a92753504a8d75471b9f49edb6fbebc898f403e4773e95feb15e80c9a99c8348d";
        let (new_index, guardians) =
            update_guardian_set::split(update_guardian_set::parse_payload(b));
        assert!(new_index == 1, 0);
        assert!(vector::length(&guardians) == 19, 0);
        let expected = vector[
            create_guardian(x"58cc3ae5c097b213ce3c81979e1b9f9570746aa5"),
            create_guardian(x"ff6cb952589bde862c25ef4392132fb9d4a42157"),
            create_guardian(x"114de8460193bdf3a2fcf81f86a09765f4762fd1"),
            create_guardian(x"107a0086b32d7a0977926a205131d8731d39cbeb"),
            create_guardian(x"8c82b2fd82faed2711d59af0f2499d16e726f6b2"),
            create_guardian(x"11b39756c042441be6d8650b69b54ebe715e2343"),
            create_guardian(x"54ce5b4d348fb74b958e8966e2ec3dbd4958a7cd"),
            create_guardian(x"eb5f7389fa26941519f0863349c223b73a6ddee7"),
            create_guardian(x"74a3bf913953d695260d88bc1aa25a4eee363ef0"),
            create_guardian(x"000ac0076727b35fbea2dac28fee5ccb0fea768e"),
            create_guardian(x"af45ced136b9d9e24903464ae889f5c8a723fc14"),
            create_guardian(x"f93124b7c738843cbb89e864c862c38cddcccf95"),
            create_guardian(x"d2cc37a4dc036a8d232b48f62cdd4731412f4890"),
            create_guardian(x"da798f6896a3331f64b48c12d1d57fd9cbe70811"),
            create_guardian(x"71aa1be1d36cafe3867910f99c09e347899c19c3"),
            create_guardian(x"8192b6e7387ccd768277c17dab1b7a5027c0b3cf"),
            create_guardian(x"178e21ad2e77ae06711549cfbb1f9c7a9d8096e8"),
            create_guardian(x"5e1487f35515d02a92753504a8d75471b9f49edb"),
            create_guardian(x"6fbebc898f403e4773e95feb15e80c9a99c8348d"),
        ];
        assert!(expected == guardians, 0);
    }

    #[test]
    public fun test_guardian_set_expiry() {
        use wormhole::state::{State, Self as worm_state};
        use wormhole::test_state::{init_wormhole_state};

        let (admin, _, _) = people();
        let test = init_wormhole_state(scenario(), admin, 0);

        next_tx(&mut test, admin);{
            let state = take_shared<State>(&test);
            let first_index = worm_state::get_current_guardian_set_index(&state);
            let guardian_set = worm_state::get_guardian_set(&state, first_index);
            // make sure guardian set is active
            assert!(
                worm_state::guardian_set_is_active(
                    &state,
                    &guardian_set,
                    ctx(&mut test)
                ),
                0
            );

            // do an upgrade
            update_guardian_set::do_upgrade_test(
                &mut state,
                1, // guardian set index
                vector[
                    create_guardian(x"71aa1be1d36cafe3867910f99c09e347899c19c3")
                ], // new guardian set
                ctx(&mut test),
            );

            // make sure old guardian set is still active
            guardian_set = worm_state::get_guardian_set(&state, first_index);
            assert!(
                worm_state::guardian_set_is_active(
                    &state,
                    &guardian_set,
                    ctx(&mut test)
                ),
                0
            );

            // fast forward time beyond expiration

            // increment by 3 epochs
            increment_epoch_number(ctx(&mut test));
            increment_epoch_number(ctx(&mut test));
            increment_epoch_number(ctx(&mut test));

            // make sure old guardian set is no longer active
            assert!(
                !worm_state::guardian_set_is_active(
                    &state,
                    &guardian_set,
                    ctx(&mut test)
                ),
                0
            );

            return_shared<State>(state);
        };

        test_scenario::end(test);
    }

}
