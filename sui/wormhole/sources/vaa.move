module wormhole::vaa {
    use std::vector::{Self};
    use sui::hash::{keccak256};
    use sui::tx_context::{TxContext};

    use wormhole::bytes::{Self};
    use wormhole::bytes32::{Self};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::guardian::{Self};
    use wormhole::guardian_set::{Self, GuardianSet};
    use wormhole::guardian_signature::{Self, GuardianSignature};
    use wormhole::state::{Self, State};

    friend wormhole::update_guardian_set;

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
        guardian_set_index: u32,
        signatures:         vector<GuardianSignature>,

        /// Body
        timestamp:          u32,
        nonce:              u32,
        emitter_chain:      u16,
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
        let cur = cursor::new(bytes);
        let version = bytes::deserialize_u8(&mut cur);
        assert!(version == 1, E_WRONG_VERSION);
        let guardian_set_index = bytes::deserialize_u32_be(&mut cur);

        let num_signatures = bytes::deserialize_u8(&mut cur);
        let signatures = vector::empty();

        let i = 0;
        while (i < num_signatures) {
            let guardian_index = bytes::deserialize_u8(&mut cur);
            let r = bytes32::from_cursor(&mut cur);
            let s = bytes32::from_cursor(&mut cur);
            let recovery_id = bytes::deserialize_u8(&mut cur);
            vector::push_back(
                &mut signatures,
                guardian_signature::new(r, s, recovery_id, guardian_index)
            );
            i = i + 1;
        };

        let body = cursor::rest(cur);
        let hash = keccak256(&keccak256(&body));

        let cur = cursor::new(body);

        let timestamp = bytes::deserialize_u32_be(&mut cur);
        let nonce = bytes::deserialize_u32_be(&mut cur);
        let emitter_chain = bytes::deserialize_u16_be(&mut cur);
        let emitter_address = external_address::deserialize(&mut cur);
        let sequence = bytes::deserialize_u64_be(&mut cur);
        let consistency_level = bytes::deserialize_u8(&mut cur);

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

    public fun get_guardian_set_index(vaa: &VAA): u32 {
         vaa.guardian_set_index
    }

    public fun get_timestamp(vaa: &VAA): u32 {
         vaa.timestamp
    }

    public fun get_payload(vaa: &VAA): vector<u8> {
         vaa.payload
    }

    public fun get_hash(vaa: &VAA): vector<u8> {
         vaa.hash
    }

    public fun get_emitter_chain(vaa: &VAA): u16 {
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
    fun verify(vaa: &VAA, set: &GuardianSet, ctx: &TxContext) {
        assert!(guardian_set::is_active(set, ctx), E_GUARDIAN_SET_EXPIRED);

        let signatures = vaa.signatures;
        let num_signatures = vector::length(&signatures);
        assert!(num_signatures >= guardian_set::quorum(set), E_NO_QUORUM);

        // Reverse to pop in increasing guardian index order.
        vector::reverse(&mut signatures);

        let guardians = guardian_set::guardians(set);
        let hash = vaa.hash;

        let i = 0;
        let last_guardian_index = 0;
        while (i < num_signatures) {
            let signature = vector::pop_back(&mut signatures);
            let guardian_index = guardian_signature::index_as_u64(&signature);

            // Ensure that the provided signatures are strictly increasing.
            // This check makes sure that no duplicate signers occur. The
            // increasing order is guaranteed by the guardians, or can always be
            // reordered by the client.
            assert!(
                i == 0 || guardian_index > last_guardian_index,
                E_NON_INCREASING_SIGNERS
            );

            // If the guardian pubkey cannot be recovered using the signature
            // and message hash, revert.
            assert!(
                guardian::verify(
                    vector::borrow(guardians, guardian_index),
                    signature,
                    hash
                ),
                E_INVALID_SIGNATURE
            );

            // Continue.
            i = i + 1;
            last_guardian_index = guardian_index;
        };
    }

    /// Parses and verifies the signatures of a VAA.
    /// NOTE: this is the only public function that returns a VAA, and it should
    /// be kept that way. This ensures that if an external module receives a
    /// `VAA`, it has been verified.
    public fun parse_and_verify(state: &mut State, bytes: vector<u8>, ctx: &TxContext): VAA {
        let vaa = parse(bytes);
        let guardian_set = state::guardian_set_at(state, &vaa.guardian_set_index);
        verify(&vaa, guardian_set, ctx);
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
    public fun assert_governance(wormhole_state: &State, vaa: &VAA) {
        let latest_guardian_set_index = state::guardian_set_index(wormhole_state);
        assert!(vaa.guardian_set_index == latest_guardian_set_index, E_OLD_GUARDIAN_SET_GOVERNANCE);
        assert!(vaa.emitter_chain == state::governance_chain(wormhole_state), E_INVALID_GOVERNANCE_CHAIN);
        assert!(vaa.emitter_address == state::governance_contract(wormhole_state), E_INVALID_GOVERNANCE_EMITTER);
    }

    /// Aborts if the VAA has already been consumed. Marks the VAA as consumed
    /// the first time around.
    /// Only to be used for core bridge messages. Protocols should implement
    /// their own replay protection.
    public(friend) fun replay_protect(state: &mut State, vaa: &VAA) {
        // this calls table::add which aborts if the key already exists
        state::set_governance_action_consumed(state, vaa.hash);
    }

}

// tests
// - do_upgrade (upgrade active guardian set to new set)

// TODO: fast forward test, check that previous guardian set gets expired
// TODO: adapt the tests from the aptos contracts test suite
#[test_only]
module wormhole::vaa_test {
    // use sui::test_scenario::{Self};
    // use sui::tx_context::{Self};


    // use wormhole::guardian::{Self};
    // use wormhole::update_guardian_set::{Self, do_upgrade_test};
    // use wormhole::state::{Self, State};
    // use wormhole::wormhole_scenario::{set_up_wormhole};
    // use wormhole::myvaa::{Self as vaa};

    // /// A test VAA signed by the first guardian set (index 0) containing guardian a single
    // /// guardian beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe
    // /// It's a governance VAA (contract upgrade), so we can test all sorts of
    // /// properties
    // const GOV_VAA: vector<u8> =
    //     x"010000000001000da16466429ee8ffb09b90ca90db8326d20cfeeae0542da9dcaaad641a5aca2d6c1fe33a5970ca84fd0ff5e6d29ef9e40404eb1a8892b509f085fc725b9e23a30100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000020b10360000000000000000000000000000000000000000000000000000000000436f7265010016d8f30e4a345ea0fa5df11daac4e1866ee368d253209cf9eda012d915a2db09e6";

    // /// Identical VAA except it's signed by guardian set 1, and double signed by
    // /// beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe
    // /// Used to test that a single guardian can't supply multiple signatures
    // const GOV_VAA_DOUBLE_SIGNED: vector<u8> =
    //     x"010000000102000da16466429ee8ffb09b90ca90db8326d20cfeeae0542da9dcaaad641a5aca2d6c1fe33a5970ca84fd0ff5e6d29ef9e40404eb1a8892b509f085fc725b9e23a301000da16466429ee8ffb09b90ca90db8326d20cfeeae0542da9dcaaad641a5aca2d6c1fe33a5970ca84fd0ff5e6d29ef9e40404eb1a8892b509f085fc725b9e23a30100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000020b10360000000000000000000000000000000000000000000000000000000000436f7265010016d8f30e4a345ea0fa5df11daac4e1866ee368d253209cf9eda012d915a2db09e6";

    // /// A test VAA signed by the second guardian set (index 1) with the following two guardians:
    // /// 0: beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe
    // /// 1: 90F8bf6A479f320ead074411a4B0e7944Ea8c9C1
    // const GOV_VAA_2: vector<u8> =
    //     x"0100000001020052da07c7ba7d58661e22922a1130e75732f454e81086330f9a5337797ee7ee9d703fd55aabc257c4d53d8ab1e471e4eb1f2767bf37cc6d3d6774e2ca3ab429eb00018c9859f14027c2a62563028a2a9bbb30464ce5b86d13728b02fb85b34761d258154bb59bad87908c9b09342efa9045d4420d289bb0144729eb368ec50c45e719010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000004cdedc90000000000000000000000000000000000000000000000000000000000436f72650100167759324e86f870265b8648ef8d5ef505b2ae99840a616081eb7adc13995204a4";

    // fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    // #[test]
    // public fun test_upgrade_guardian() {
    //     let (admin, caller, _) = people();
    //     let my_scenario = test_scenario::begin(admin);
    //     let scenario = &mut my_scenario;

    //     // Initialize Wormhole.
    //     set_up_wormhole(scenario, admin, 0);

    //     // Proceed as some other transaction executor `caller`.
    //     test_scenario::next_tx(scenario, caller);

    //     {
    //         let worm_state = test_scenario::take_shared<State>(scenario);
    //         let new_guardians =
    //             vector[
    //                 guardian::new(
    //                     x"71aa1be1d36cafe3867910f99c09e347899c19c4"
    //                 )
    //             ];
    //         // upgrade guardian set
    //         // TODO: we should use a VAA to do this.
    //         do_upgrade_test(
    //             &mut worm_state,
    //             1, new_guardians, test_scenario::ctx(scenario));
    //         assert!(state::guardian_set_index(&state) == 1, 0);

    //         test_scenario::return_shared<State>(state);
    //     };

    //     // TODO: should test that a Wormhole message signed by the new guardian
    //     // set passes verification.

    //     // Done.
    //     test_scenario::end(my_scenario);
    // }

    // #[test]
    // /// Ensures that the GOV_VAA can still be verified after the guardian set
    // /// upgrade before expiry
    // public fun test_guardian_set_not_expired() {
    //     let (admin, _, _) = people();
    //     let test = init_wormhole_state(scenario(), admin, 0);

    //     next_tx(scenario, admin);{
    //         let state = take_shared<State>(scenario);

    //         // do an upgrade
    //         update_guardian_set::do_upgrade_test(
    //             &mut state,
    //             1, // guardian set index
    //             vector[
    //                 guardian::new(x"71aa1be1d36cafe3867910f99c09e347899c19c3")
    //             ],
    //             ctx(scenario)
    //         );

    //         // fast forward time before expiration
    //         increment_epoch_number(ctx(scenario));

    //         // we still expect this to verify
    //         vaa::destroy(
    //             vaa::parse_and_verify(&mut state, GOV_VAA, ctx(scenario))
    //         );
    //         return_shared<State>(state);
    //     };
    //     test_scenario::end(my_scenario);
    // }

    // #[test]
    // #[expected_failure(abort_code = vaa::E_GUARDIAN_SET_EXPIRED)]
    // /// Ensures that the GOV_VAA can no longer be verified after the guardian set
    // /// upgrade after expiry
    // public fun test_guardian_set_expired() {
    //     let (admin, _, _) = people();
    //     let test = init_wormhole_state(scenario(), admin, 0);

    //     next_tx(scenario, admin);{
    //         let state = take_shared<State>(scenario);

    //         // do an upgrade
    //         update_guardian_set::do_upgrade_test(
    //             &mut state,
    //             1, // guardian set index
    //             vector[
    //                 guardian::new(x"71aa1be1d36cafe3867910f99c09e347899c19c3")
    //             ],
    //             ctx(scenario)
    //         );

    //         // fast forward time beyond expiration
    //         increment_epoch_number(ctx(scenario));
    //         increment_epoch_number(ctx(scenario));
    //         increment_epoch_number(ctx(scenario));

    //         // we expect this to fail because guardian set has expired
    //         vaa::destroy(
    //             vaa::parse_and_verify(&mut state, GOV_VAA, ctx(scenario))
    //         );
    //         return_shared<State>(state);
    //     };
    //     test_scenario::end(my_scenario);
    // }

    // #[test]
    // #[expected_failure(abort_code = vaa::E_OLD_GUARDIAN_SET_GOVERNANCE)]
    // /// Ensures that governance GOV_VAAs can only be verified by the latest guardian
    // /// set, even if the signer hasn't expired yet
    // public fun test_governance_guardian_set_latest() {
    //     let (admin, _, _) = people();
    //     let test = init_wormhole_state(scenario(), admin, 0);

    //     next_tx(scenario, admin);{
    //         let state = take_shared<State>(scenario);

    //         // do an upgrade
    //         update_guardian_set::do_upgrade_test(
    //             &mut state,
    //             1, // guardian set index
    //             vector[
    //                 guardian::new(x"71aa1be1d36cafe3867910f99c09e347899c19c3")
    //             ],
    //             ctx(scenario)
    //         );

    //         // fast forward time before expiration
    //         increment_epoch_number(ctx(scenario));

    //         //still expect this to verify
    //         let vaa =
    //             vaa::parse_and_verify(&mut state, GOV_VAA, ctx(scenario));

    //         // expect this to fail
    //         vaa::assert_governance(&mut state, &vaa);

    //         vaa::destroy(vaa);

    //         return_shared<State>(state);
    //     };
    //     test_scenario::end(my_scenario);
    // }

    // #[test]
    // #[expected_failure(abort_code = vaa::E_INVALID_GOVERNANCE_EMITTER)]
    // /// Ensures that governance GOV_VAAs can only be sent from the correct
    // /// governance emitter
    // public fun test_invalid_governance_emitter() {
    //     let (admin, _, _) = people();
    //     let test = init_wormhole_state(scenario(), admin, 0);

    //     next_tx(scenario, admin);{
    //         let state = take_shared<State>(scenario);
    //         state::set_governance_contract(
    //             &mut state,
    //             x"0000000000000000000000000000000000000000000000000000000000000005"
    //         ); // set emitter contract to wrong contract

    //         // expect this to succeed
    //         let vaa =
    //             vaa::parse_and_verify(&mut state, GOV_VAA, ctx(scenario));

    //         // expect this to fail
    //         vaa::assert_governance(&mut state, &vaa);

    //         vaa::destroy(vaa);

    //         return_shared<State>(state);
    //     };
    //     test_scenario::end(my_scenario);
    // }

    // #[test]
    // #[expected_failure(abort_code = vaa::E_INVALID_GOVERNANCE_CHAIN)]
    // /// Ensures that governance GOV_VAAs can only be sent from the correct
    // /// governance chain
    // public fun test_invalid_governance_chain() {
    //     let (admin, _, _) = people();
    //     let test = init_wormhole_state(scenario(), admin, 0);

    //     next_tx(scenario, admin);{
    //         let state = take_shared<State>(scenario);
    //         state::set_governance_chain(&mut state, 200); // set governance chain to wrong chain

    //         // expect this to succeed
    //         let vaa =
    //             vaa::parse_and_verify(&mut state, GOV_VAA, ctx(scenario));

    //         // expect this to fail
    //         vaa::assert_governance(&mut state, &vaa);

    //         vaa::destroy(vaa);

    //         return_shared<State>(state);
    //     };
    //     test_scenario::end(my_scenario);
    // }

    // #[test]
    // public fun test_quorum() {
    //     let (admin, _, _) = people();
    //     let test = init_wormhole_state(scenario(), admin, 0);

    //     next_tx(scenario, admin);{
    //         let state = take_shared<State>(scenario);

    //         // do an upgrade
    //         update_guardian_set::do_upgrade_test(
    //             &mut state,
    //             1, // guardian set index
    //             vector[
    //                 guardian::new(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"),
    //                 guardian::new(x"90F8bf6A479f320ead074411a4B0e7944Ea8c9C1")
    //             ],
    //             ctx(scenario),
    //         );

    //         // we expect this to succeed because both guardians signed in the correct order
    //         vaa::destroy(
    //             vaa::parse_and_verify(&mut state, GOV_VAA_2, ctx(scenario))
    //         );
    //         return_shared<State>(state);
    //     };
    //     test_scenario::end(my_scenario);
    // }

    // #[test]
    // #[expected_failure(abort_code = vaa::E_NO_QUORUM)]
    // public fun test_no_quorum() {
    //     let (admin, _, _) = people();
    //     let test = init_wormhole_state(scenario(), admin, 0);

    //     next_tx(scenario, admin);{
    //         let state = take_shared<State>(scenario);

    //         // do an upgrade
    //         update_guardian_set::do_upgrade_test(
    //             &mut state,
    //             1, // guardian set index
    //             vector[
    //                 guardian::new(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"),
    //                 guardian::new(x"90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"),
    //                 guardian::new(x"5e1487f35515d02a92753504a8d75471b9f49edb")
    //             ],
    //             ctx(scenario),
    //         );

    //         // we expect this to fail because not enough signatures
    //         vaa::destroy(vaa::parse_and_verify(&mut state, GOV_VAA_2, ctx(scenario)));
    //         return_shared<State>(state);
    //     };
    //     test_scenario::end(my_scenario);
    // }

    // #[test]
    // #[expected_failure(abort_code = vaa::E_NON_INCREASING_SIGNERS)]
    // public fun test_double_signed() {
    //     let (admin, _, _) = people();
    //     let test = init_wormhole_state(scenario(), admin, 0);

    //     next_tx(scenario, admin);{
    //         let state = take_shared<State>(scenario);

    //         // do an upgrade
    //         update_guardian_set::do_upgrade_test(
    //             &mut state,
    //             1, // guardian set index
    //             vector[
    //                 guardian::new(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"),
    //                 guardian::new(x"90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"),
    //             ],
    //             ctx(scenario),
    //         );

    //         // we expect this to fail because
    //         // beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe signed this twice
    //         vaa::destroy(
    //             vaa::parse_and_verify(
    //                 &mut state,
    //                 GOV_VAA_DOUBLE_SIGNED,
    //                 ctx(scenario)
    //             )
    //         );
    //         return_shared<State>(state);
    //     };
    //     test_scenario::end(my_scenario);
    // }

    // #[test]
    // #[expected_failure(abort_code = vaa::E_INVALID_SIGNATURE)]
    // public fun test_out_of_order_signers() {
    //     let (admin, _, _) = people();
    //     let test = init_wormhole_state(scenario(), admin, 0);

    //     next_tx(scenario, admin);{
    //         let state = take_shared<State>(scenario);

    //         // do an upgrade
    //         update_guardian_set::do_upgrade_test(
    //             &mut state,
    //             1, // guardian set index
    //             vector[
    //                 // guardians are set up in opposite order
    //                 guardian::new(x"90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"),
    //                 guardian::new(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"),
    //             ],
    //             ctx(scenario),
    //         );

    //         // we expect this to fail because signatures are out of order
    //         vaa::destroy(
    //             vaa::parse_and_verify(&mut state, GOV_VAA_2, ctx(scenario))
    //         );
    //         return_shared<State>(state);
    //     };
    //     test_scenario::end(my_scenario);
    // }

}
