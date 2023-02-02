module wormhole::myvaa {
    use std::vector;
    use sui::tx_context::TxContext;
    //use 0x1::secp256k1;

    use wormhole::myu16::{U16};
    use wormhole::myu32::{U32};
    use wormhole::deserialize;
    use wormhole::cursor;
    use wormhole::guardian_pubkey;
    use wormhole::structs::{
        Guardian,
        GuardianSet,
        Signature,
        create_signature,
        get_guardians,
        unpack_signature,
        get_address,
    };
    use wormhole::state::{Self, State};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::keccak256::keccak256;

    friend wormhole::guardian_set_upgrade;
    //friend wormhole::contract_upgrade;

    const E_NO_QUORUM: u64 = 0x0;
    const E_TOO_MANY_SIGNATURES: u64 = 0x1;
    const E_INVALID_SIGNATURE: u64 = 0x2;
    const E_GUARDIAN_SET_EXPIRED: u64 = 0x3;
    const E_INVALID_GOVERNANCE_CHAIN: u64 = 0x4;
    const E_INVALID_GOVERNANCE_EMITTER: u64 = 0x5;
    const E_WRONG_VERSION: u64 = 0x6;
    const E_NON_INCREASING_SIGNERS: u64 = 0x7;
    const E_OLD_GUARDIAN_SET_GOVERNANCE: u64 = 0x8;

    struct VAA {
        /// Header
        guardian_set_index: U32,
        signatures:         vector<Signature>,

        /// Body
        timestamp:          U32,
        nonce:              U32,
        emitter_chain:      U16,
        emitter_address:    ExternalAddress,
        sequence:           u64,
        consistency_level:  u8,
        hash:               vector<u8>, // 32 bytes
        payload:            vector<u8>, // variable bytes
    }

    //break

    #[test_only]
    public fun parse_test(bytes: vector<u8>): VAA {
        parse(bytes)
    }

    /// Parses a VAA.
    /// Does not do any verification, and is thus private.
    /// This ensures the invariant that if an external module receives a `VAA`
    /// object, its signatures must have been verified, because the only public
    /// function that returns a VAA is `parse_and_verify`
    fun parse(bytes: vector<u8>): VAA {
        let cur = cursor::cursor_init(bytes);
        let version = deserialize::deserialize_u8(&mut cur);
        assert!(version == 1, E_WRONG_VERSION);
        let guardian_set_index = deserialize::deserialize_u32(&mut cur);

        let signatures_len = deserialize::deserialize_u8(&mut cur);
        let signatures = vector::empty<Signature>();

        while (signatures_len > 0) {
            let guardian_index = deserialize::deserialize_u8(&mut cur);
            let sig = deserialize::deserialize_vector(&mut cur, 64);
            let recovery_id = deserialize::deserialize_u8(&mut cur);
            vector::push_back(&mut signatures, create_signature(sig, recovery_id, guardian_index));
            signatures_len = signatures_len - 1;
        };

        let body = cursor::rest(cur);
        let hash = keccak256(keccak256(body));

        let cur = cursor::cursor_init(body);

        let timestamp = deserialize::deserialize_u32(&mut cur);
        let nonce = deserialize::deserialize_u32(&mut cur);
        let emitter_chain = deserialize::deserialize_u16(&mut cur);
        let emitter_address = external_address::deserialize(&mut cur);
        let sequence = deserialize::deserialize_u64(&mut cur);
        let consistency_level = deserialize::deserialize_u8(&mut cur);

        let payload = cursor::rest(cur);

        VAA {
            guardian_set_index,
            signatures,
            timestamp,
            nonce,
            emitter_chain,
            emitter_address,
            sequence,
            consistency_level,
            hash,
            payload,
        }
    }

    public fun get_guardian_set_index(vaa: &VAA): U32 {
         vaa.guardian_set_index
    }

    public fun get_timestamp(vaa: &VAA): U32 {
         vaa.timestamp
    }

    public fun get_payload(vaa: &VAA): vector<u8> {
         vaa.payload
    }

    public fun get_hash(vaa: &VAA): vector<u8> {
         vaa.hash
    }

    public fun get_emitter_chain(vaa: &VAA): U16 {
         vaa.emitter_chain
    }

    public fun get_emitter_address(vaa: &VAA): ExternalAddress {
         vaa.emitter_address
    }

    public fun get_sequence(vaa: &VAA): u64 {
         vaa.sequence
    }

    public fun get_consistency_level(vaa: &VAA): u8 {
        vaa.consistency_level
    }

    //  break

    public fun destroy(vaa: VAA): vector<u8> {
         let VAA {
            guardian_set_index: _,
            signatures: _,
            timestamp: _,
            nonce: _,
            emitter_chain: _,
            emitter_address: _,
            sequence: _,
            consistency_level: _,
            hash: _,
            payload,
         } = vaa;
        payload
    }

    /// Verifies the signatures of a VAA.
    /// It's private, because there's no point calling it externally, since VAAs
    /// external to this module have already been verified (by construction).
    fun verify(vaa: &VAA, state: &State, guardian_set: &GuardianSet, ctx: &TxContext) {
        assert!(state::guardian_set_is_active(state, guardian_set, ctx), E_GUARDIAN_SET_EXPIRED);

        let guardians = get_guardians(guardian_set);
        let hash = vaa.hash;
        let sigs_len = vector::length<Signature>(&vaa.signatures);
        let guardians_len = vector::length<Guardian>(&guardians);

        assert!(sigs_len >= quorum(guardians_len), E_NO_QUORUM);

        let sig_i = 0;
        let last_index = 0;
        while (sig_i < sigs_len) {
            let (sig, recovery_id, guardian_index) = unpack_signature(vector::borrow(&vaa.signatures, sig_i));

            // Ensure that the provided signatures are strictly increasing.
            // This check makes sure that no duplicate signers occur. The
            // increasing order is guaranteed by the guardians, or can always be
            // reordered by the client.
            assert!(sig_i == 0 || guardian_index > last_index, E_NON_INCREASING_SIGNERS);
            last_index = guardian_index;

            let address = guardian_pubkey::from_signature(hash, recovery_id, sig);

            let cur_guardian = vector::borrow<Guardian>(&guardians, (guardian_index as u64));
            let cur_address = get_address(cur_guardian);

            assert!(address == cur_address, E_INVALID_SIGNATURE);

            sig_i = sig_i + 1;
        };
    }

    /// Parses and verifies the signatures of a VAA.
    /// NOTE: this is the only public function that returns a VAA, and it should
    /// be kept that way. This ensures that if an external module receives a
    /// `VAA`, it has been verified.
    public fun parse_and_verify(state: &mut State, bytes: vector<u8>, ctx: &TxContext): VAA {
        let vaa = parse(bytes);
        let guardian_set = state::get_guardian_set(state, vaa.guardian_set_index);
        verify(&vaa, state, &guardian_set, ctx);
        vaa
    }

    /// Gets a VAA payload without doing verififcation on the VAA. This method is
    /// used for convenience in the Coin package, for example, for creating new tokens
    /// with asset metadata in a token attestation VAA payload.
    public fun parse_and_get_payload(bytes: vector<u8>): vector<u8> {
        let vaa = parse(bytes);
        let payload = destroy(vaa);
        return payload
    }

    /// Aborts if the VAA is not governance (i.e. sent from the governance
    /// emitter on the governance chain)
    public fun assert_governance(state: &State, vaa: &VAA) {
        let latest_guardian_set_index = state::get_current_guardian_set_index(state);
        assert!(vaa.guardian_set_index == latest_guardian_set_index, E_OLD_GUARDIAN_SET_GOVERNANCE);
        assert!(vaa.emitter_chain == state::get_governance_chain(state), E_INVALID_GOVERNANCE_CHAIN);
        assert!(vaa.emitter_address == state::get_governance_contract(state), E_INVALID_GOVERNANCE_EMITTER);
    }

    /// Aborts if the VAA has already been consumed. Marks the VAA as consumed
    /// the first time around.
    /// Only to be used for core bridge messages. Protocols should implement
    /// their own replay protection.
    public(friend) fun replay_protect(state: &mut State, vaa: &VAA) {
        // this calls table::add which aborts if the key already exists
        state::set_governance_action_consumed(state, vaa.hash);
    }

    /// Returns the minimum number of signatures required for a VAA to be valid.
    public fun quorum(num_guardians: u64): u64 {
        (num_guardians * 2) / 3 + 1
    }

}

// tests
// - do_upgrade (upgrade active guardian set to new set)

// TODO: fast forward test, check that previous guardian set gets expired
// TODO: adapt the tests from the aptos contracts test suite
#[test_only]
module wormhole::vaa_test {
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_shared, return_shared};
    use sui::tx_context::{increment_epoch_number};

    fun scenario(): Scenario { test_scenario::begin(@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    use wormhole::guardian_set_upgrade::{Self, do_upgrade_test};
    use wormhole::state::{Self, State};
    use wormhole::test_state::{init_wormhole_state};
    use wormhole::structs::{Self, create_guardian};
    use wormhole::myu32::{Self as u32};
    use wormhole::myvaa::{Self as vaa};

    /// A test VAA signed by the first guardian set (index 0) containing guardian a single
    /// guardian beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe
    /// It's a governance VAA (contract upgrade), so we can test all sorts of
    /// properties
    const GOV_VAA: vector<u8> = x"010000000001000da16466429ee8ffb09b90ca90db8326d20cfeeae0542da9dcaaad641a5aca2d6c1fe33a5970ca84fd0ff5e6d29ef9e40404eb1a8892b509f085fc725b9e23a30100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000020b10360000000000000000000000000000000000000000000000000000000000436f7265010016d8f30e4a345ea0fa5df11daac4e1866ee368d253209cf9eda012d915a2db09e6";

    /// Identical VAA except it's signed by guardian set 1, and double signed by
    /// beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe
    /// Used to test that a single guardian can't supply multiple signatures
    const GOV_VAA_DOUBLE_SIGNED: vector<u8> = x"010000000102000da16466429ee8ffb09b90ca90db8326d20cfeeae0542da9dcaaad641a5aca2d6c1fe33a5970ca84fd0ff5e6d29ef9e40404eb1a8892b509f085fc725b9e23a301000da16466429ee8ffb09b90ca90db8326d20cfeeae0542da9dcaaad641a5aca2d6c1fe33a5970ca84fd0ff5e6d29ef9e40404eb1a8892b509f085fc725b9e23a30100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000020b10360000000000000000000000000000000000000000000000000000000000436f7265010016d8f30e4a345ea0fa5df11daac4e1866ee368d253209cf9eda012d915a2db09e6";

    /// A test VAA signed by the second guardian set (index 1) with the following two guardians:
    /// 0: beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe
    /// 1: 90F8bf6A479f320ead074411a4B0e7944Ea8c9C1
    const GOV_VAA_2: vector<u8> = x"0100000001020052da07c7ba7d58661e22922a1130e75732f454e81086330f9a5337797ee7ee9d703fd55aabc257c4d53d8ab1e471e4eb1f2767bf37cc6d3d6774e2ca3ab429eb00018c9859f14027c2a62563028a2a9bbb30464ce5b86d13728b02fb85b34761d258154bb59bad87908c9b09342efa9045d4420d289bb0144729eb368ec50c45e719010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000004cdedc90000000000000000000000000000000000000000000000000000000000436f72650100167759324e86f870265b8648ef8d5ef505b2ae99840a616081eb7adc13995204a4";

    #[test]
    fun test_upgrade_guardian() {
        test_upgrade_guardian_(scenario())
    }

    fun test_upgrade_guardian_(test: Scenario) {
        let (admin, _, _) = people();
        test = init_wormhole_state(test, admin, 0);
        next_tx(&mut test, admin);{
            let state = take_shared<State>(&mut test);
            let new_guardians = vector[structs::create_guardian(x"71aa1be1d36cafe3867910f99c09e347899c19c4")];
            // upgrade guardian set
            do_upgrade_test(&mut state, u32::from_u64(1), new_guardians, ctx(&mut test));
            assert!(state::get_current_guardian_set_index(&state)==u32::from_u64(1), 0);
            return_shared<State>(state);
        };
        test_scenario::end(test);
    }

    #[test]
    /// Ensures that the GOV_VAA can still be verified after the guardian set
    /// upgrade before expiry
    public fun test_guardian_set_not_expired() {
        let (admin, _, _) = people();
        let test = init_wormhole_state(scenario(), admin, 0);

        next_tx(&mut test, admin);{
            let state = take_shared<State>(&test);

            // do an upgrade
            guardian_set_upgrade::do_upgrade_test(
                &mut state,
                u32::from_u64(1),
                vector[create_guardian(x"71aa1be1d36cafe3867910f99c09e347899c19c3")],
                ctx(&mut test)
            );

            // fast forward time before expiration
            increment_epoch_number(ctx(&mut test));

            // we still expect this to verify
            vaa::destroy(vaa::parse_and_verify(&mut state, GOV_VAA, ctx(&mut test)));
            return_shared<State>(state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = vaa::E_GUARDIAN_SET_EXPIRED)]
    /// Ensures that the GOV_VAA can no longer be verified after the guardian set
    /// upgrade after expiry
    public fun test_guardian_set_expired() {
        let (admin, _, _) = people();
        let test = init_wormhole_state(scenario(), admin, 0);

        next_tx(&mut test, admin);{
            let state = take_shared<State>(&test);

            // do an upgrade
            guardian_set_upgrade::do_upgrade_test(
                &mut state,
                u32::from_u64(1),
                vector[create_guardian(x"71aa1be1d36cafe3867910f99c09e347899c19c3")],
                ctx(&mut test)
            );

            // fast forward time beyond expiration
            increment_epoch_number(ctx(&mut test));
            increment_epoch_number(ctx(&mut test));
            increment_epoch_number(ctx(&mut test));

            // we expect this to fail because guardian set has expired
            vaa::destroy(vaa::parse_and_verify(&mut state, GOV_VAA, ctx(&mut test)));
            return_shared<State>(state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = vaa::E_OLD_GUARDIAN_SET_GOVERNANCE)]
    /// Ensures that governance GOV_VAAs can only be verified by the latest guardian
    /// set, even if the signer hasn't expired yet
    public fun test_governance_guardian_set_latest() {
        let (admin, _, _) = people();
        let test = init_wormhole_state(scenario(), admin, 0);

        next_tx(&mut test, admin);{
            let state = take_shared<State>(&test);

            // do an upgrade
            guardian_set_upgrade::do_upgrade_test(
                &mut state,
                u32::from_u64(1),
                vector[create_guardian(x"71aa1be1d36cafe3867910f99c09e347899c19c3")],
                ctx(&mut test)
            );

            // fast forward time before expiration
            increment_epoch_number(ctx(&mut test));

            //still expect this to verify
            let vaa = vaa::parse_and_verify(&mut state, GOV_VAA, ctx(&mut test));

            // expect this to fail
            vaa::assert_governance(&mut state, &vaa);

            vaa::destroy(vaa);

            return_shared<State>(state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = vaa::E_INVALID_GOVERNANCE_EMITTER)]
    /// Ensures that governance GOV_VAAs can only be sent from the correct governance emitter
    public fun test_invalid_governance_emitter() {
        let (admin, _, _) = people();
        let test = init_wormhole_state(scenario(), admin, 0);

        next_tx(&mut test, admin);{
            let state = take_shared<State>(&test);
            state::set_governance_contract(&mut state,  x"0000000000000000000000000000000000000000000000000000000000000005"); // set emitter contract to wrong contract

            // expect this to succeed
            let vaa = vaa::parse_and_verify(&mut state, GOV_VAA, ctx(&mut test));

            // expect this to fail
            vaa::assert_governance(&mut state, &vaa);

            vaa::destroy(vaa);

            return_shared<State>(state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = vaa::E_INVALID_GOVERNANCE_CHAIN)]
    /// Ensures that governance GOV_VAAs can only be sent from the correct governance chain
    public fun test_invalid_governance_chain() {
        let (admin, _, _) = people();
        let test = init_wormhole_state(scenario(), admin, 0);

        next_tx(&mut test, admin);{
            let state = take_shared<State>(&test);
            state::set_governance_chain_id(&mut state,  200); // set governance chain to wrong chain

            // expect this to succeed
            let vaa = vaa::parse_and_verify(&mut state, GOV_VAA, ctx(&mut test));

            // expect this to fail
            vaa::assert_governance(&mut state, &vaa);

            vaa::destroy(vaa);

            return_shared<State>(state);
        };
        test_scenario::end(test);
    }

    #[test]
    public fun test_quorum() {
        let (admin, _, _) = people();
        let test = init_wormhole_state(scenario(), admin, 0);

        next_tx(&mut test, admin);{
            let state = take_shared<State>(&test);

            // do an upgrade
            guardian_set_upgrade::do_upgrade_test(
                &mut state,
                u32::from_u64(1),
                vector[
                    create_guardian(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"),
                    create_guardian(x"90F8bf6A479f320ead074411a4B0e7944Ea8c9C1")
                ],
                ctx(&mut test),
            );

            // we expect this to succeed because both guardians signed in the correct order
            vaa::destroy(vaa::parse_and_verify(&mut state, GOV_VAA_2, ctx(&mut test)));
            return_shared<State>(state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = vaa::E_NO_QUORUM)]
    public fun test_no_quorum() {
        let (admin, _, _) = people();
        let test = init_wormhole_state(scenario(), admin, 0);

        next_tx(&mut test, admin);{
            let state = take_shared<State>(&test);

            // do an upgrade
            guardian_set_upgrade::do_upgrade_test(
                &mut state,
                u32::from_u64(1),
                vector[
                    create_guardian(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"),
                    create_guardian(x"90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"),
                    create_guardian(x"5e1487f35515d02a92753504a8d75471b9f49edb")
                ],
                ctx(&mut test),
            );

            // we expect this to fail because not enough signatures
            vaa::destroy(vaa::parse_and_verify(&mut state, GOV_VAA_2, ctx(&mut test)));
            return_shared<State>(state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = vaa::E_NON_INCREASING_SIGNERS)]
    public fun test_double_signed() {
        let (admin, _, _) = people();
        let test = init_wormhole_state(scenario(), admin, 0);

        next_tx(&mut test, admin);{
            let state = take_shared<State>(&test);

            // do an upgrade
            guardian_set_upgrade::do_upgrade_test(
                &mut state,
                u32::from_u64(1),
                vector[
                    create_guardian(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"),
                    create_guardian(x"90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"),
                ],
                ctx(&mut test),
            );

            // we expect this to fail because
            // beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe signed this twice
            vaa::destroy(vaa::parse_and_verify(&mut state, GOV_VAA_DOUBLE_SIGNED, ctx(&mut test)));
            return_shared<State>(state);
        };
        test_scenario::end(test);
    }

    #[test]
    #[expected_failure(abort_code = vaa::E_INVALID_SIGNATURE)]
    public fun test_out_of_order_signers() {
        let (admin, _, _) = people();
        let test = init_wormhole_state(scenario(), admin, 0);

        next_tx(&mut test, admin);{
            let state = take_shared<State>(&test);

            // do an upgrade
            guardian_set_upgrade::do_upgrade_test(
                &mut state,
                u32::from_u64(1),
                vector[
                    // guardians are set up in opposite order
                    create_guardian(x"90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"),
                    create_guardian(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"),
                ],
                ctx(&mut test),
            );

            // we expect this to fail because signatures are out of order
            vaa::destroy(vaa::parse_and_verify(&mut state, GOV_VAA_2, ctx(&mut test)));
            return_shared<State>(state);
        };
        test_scenario::end(test);
    }

}
