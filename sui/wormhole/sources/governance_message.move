// SPDX-License-Identifier: Apache 2

/// This module implements a custom type representing a Guardian governance
/// action. Each governance action has an associated module name, relevant chain
/// and payload encoding instructions/data used to perform an adminstrative
/// change on a contract.
module wormhole::governance_message {
    use wormhole::bytes::{Self};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::cursor::{Self};
    use wormhole::state::{Self, State, chain_id};
    use wormhole::vaa::{Self, VAA};

    /// Guardian set used to sign VAA did not use current Guardian set.
    const E_OLD_GUARDIAN_SET_GOVERNANCE: u64 = 0;
    /// Governance chain disagrees with what is stored in Wormhole `State`.
    const E_INVALID_GOVERNANCE_CHAIN: u64 = 1;
    /// Governance emitter address disagrees with what is stored in Wormhole
    /// `State`.
    const E_INVALID_GOVERNANCE_EMITTER: u64 = 2;
    /// Governance module name does not match.
    const E_INVALID_GOVERNANCE_MODULE: u64 = 4;
    /// Governance action does not match.
    const E_INVALID_GOVERNANCE_ACTION: u64 = 5;
    /// Governance target chain not indicative of global action.
    const E_GOVERNANCE_TARGET_CHAIN_NONZERO: u64 = 6;
    /// Governance target chain not indicative of actino specifically for Sui
    /// Wormhole contract.
    const E_GOVERNANCE_TARGET_CHAIN_NOT_SUI: u64 = 7;

    /// Deserialized governance decree from `VAA` payload.
    struct GovernanceMessage {
        module_name: Bytes32,
        action: u8,
        chain: u16,
        payload: vector<u8>,
        vaa_hash: Bytes32
    }

    /// Retrieve governance module name.
    public fun module_name(self: &GovernanceMessage): Bytes32 {
        self.module_name
    }

    /// Retrieve governance action (i.e. payload ID).
    public fun action(self: &GovernanceMessage): u8 {
        self.action
    }

    /// A.K.A. target chain == 0.
    public fun is_global_action(self: &GovernanceMessage): bool {
        self.chain == 0
    }

    /// A.K.A. target chain == `wormhole::state::chain_id()`.
    public fun is_local_action(self: &GovernanceMessage): bool {
        self.chain == chain_id()
    }

    /// Computed keccak256 hash of `VAA` message body.
    public fun vaa_hash(self: &GovernanceMessage): Bytes32 {
        self.vaa_hash
    }

    /// Destroy `GovernanceMessage` to take governance payload.
    public fun take_payload(msg: GovernanceMessage): vector<u8> {
        let GovernanceMessage {
            module_name: _,
            action: _,
            chain: _,
            vaa_hash: _,
            payload
        } = msg;

        payload
    }

    /// Passing in a deserialized `VAA`, Wormhole performs additional governance
    /// checks to validate governance emitter before returning deserialized
    /// `GovernanceMessage`.
    ///
    /// NOTE: It is expected that these VAAs are consumed only once using
    /// `ConsumedVAAs`. Those contracts that use Guardian governance to perform
    /// administrative functions are expected to have this container to protect
    /// against replaying these governance actions.
    public fun verify_vaa(
        wormhole_state: &State,
        verified_vaa: VAA,
    ): GovernanceMessage {
        // This VAA must have originated from the governance emitter.
        assert_governance_emitter(wormhole_state, &verified_vaa);

        // Cache VAA digest.
        let vaa_hash = vaa::digest(&verified_vaa);

        // Finally deserialize Wormhole payload as governance message.
        let cur = cursor::new(vaa::take_payload(verified_vaa));
        let module_name = bytes32::take_bytes(&mut cur);
        let action = bytes::take_u8(&mut cur);
        let chain = bytes::take_u16_be(&mut cur);
        let payload = cursor::take_rest(cur);

        GovernanceMessage { module_name, action, chain, payload, vaa_hash }
    }

    /// Check module name, action and whether this action is intended for all
    /// chains before `take_payload` is called.
    ///
    /// NOTE: It is expected that these governance messages are consumed only
    /// once using `ConsumedVAAs` to store the VAA digest. Those contracts that
    /// use Guardian governance to perform administrative functions are expected
    /// to have this container to protect against replaying these governance
    /// actions.
    public fun take_global_action(
        msg: GovernanceMessage,
        expected_module_name: Bytes32,
        expected_action: u8
    ): vector<u8> {
        assert_module_and_action(&msg, expected_module_name, expected_action);

        // New guardian sets are applied to all Wormhole contracts.
        assert!(is_global_action(&msg), E_GOVERNANCE_TARGET_CHAIN_NONZERO);

        take_payload(msg)
    }

    /// Check module name, action and whether this action is intended for Sui's
    /// chain ID before `take_payload` is called.
    ///
    /// NOTE: It is expected that these governance messages are consumed only
    /// once using `ConsumedVAAs` to store the VAA digest. Those contracts that
    /// use Guardian governance to perform administrative functions are expected
    /// to have this container to protect against replaying these governance
    /// actions.
    public fun take_local_action(
        msg: GovernanceMessage,
        expected_module_name: Bytes32,
        expected_action: u8
    ): vector<u8> {
        assert_module_and_action(&msg, expected_module_name, expected_action);

        // New guardian sets are applied to all Wormhole contracts.
        assert!(is_local_action(&msg), E_GOVERNANCE_TARGET_CHAIN_NOT_SUI);

        take_payload(msg)
    }

    fun assert_module_and_action(
        self: &GovernanceMessage,
        expected_module_name: Bytes32,
        expected_action: u8
    ) {
        // Governance action must be for Wormhole (Core Bridge).
        assert!(
            self.module_name == expected_module_name,
            E_INVALID_GOVERNANCE_MODULE
        );

        // Action must be specifically to update the guardian set.
        assert!(
            self.action == expected_action,
            E_INVALID_GOVERNANCE_ACTION
        );
    }

    #[test_only]
    public fun assert_module_and_action_test_only(
        self: &GovernanceMessage,
        expected_module_name: Bytes32,
        expected_action: u8
    ) {
        assert_module_and_action(self, expected_module_name, expected_action)
    }

    /// Aborts if the VAA is not governance (i.e. sent from the governance
    /// emitter on the governance chain)
    fun assert_governance_emitter(wormhole_state: &State, verified_vaa: &VAA) {
        // This state capability ensures that the current build version is used.
        state::assert_current(wormhole_state);

        // Protect against governance actions enacted using an old guardian set.
        // This is not a protection found in the other Wormhole contracts.
        assert!(
            vaa::guardian_set_index(verified_vaa) == state::guardian_set_index(wormhole_state),
            E_OLD_GUARDIAN_SET_GOVERNANCE
        );

        // Both the emitter chain and address must equal those known by the
        // Wormhole `State`.
        assert!(
            vaa::emitter_chain(verified_vaa) == state::governance_chain(wormhole_state),
            E_INVALID_GOVERNANCE_CHAIN
        );
        assert!(
            vaa::emitter_address(verified_vaa) == state::governance_contract(wormhole_state),
            E_INVALID_GOVERNANCE_EMITTER
        );
    }

    #[test_only]
    public fun assert_governance_emitter_test_only(
        wormhole_state: &State,
        verified_vaa: &VAA
    ) {
        assert_governance_emitter(wormhole_state, verified_vaa)
    }

    #[test_only]
    public fun payload(self: &GovernanceMessage): vector<u8> {
        self.payload
    }

    #[test_only]
    public fun destroy(msg: GovernanceMessage) {
        take_payload(msg);
    }
}

#[test_only]
module wormhole::governance_message_tests {
    use sui::test_scenario::{Self};

    use wormhole::bytes32::{Self};
    use wormhole::state::{Self};
    use wormhole::governance_message::{Self};
    use wormhole::vaa::{Self};
    use wormhole::version_control::{Self, V__0_1_0, V__MIGRATED};
    use wormhole::wormhole_scenario::{
        set_up_wormhole,
        person,
        return_clock,
        return_state,
        take_clock,
        take_state
    };

    const VAA_UPDATE_GUARDIAN_SET_1: vector<u8> =
        x"010000000001004f74e9596bd8246ef456918594ae16e81365b52c0cf4490b2a029fb101b058311f4a5592baeac014dc58215faad36453467a85a4c3e1c6cf5166e80f6e4dc50b0100bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f72650200000000000113befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe88d7d8b32a9105d228100e72dffe2fae0705d31c58076f561cc62a47087b567c86f986426dfcd000bd6e9833490f8fa87c733a183cd076a6cbd29074b853fcf0a5c78c1b56d15fce7a154e6ebe9ed7a2af3503dbd2e37518ab04d7ce78b630f98b15b78a785632dea5609064803b1c8ea8bb2c77a6004bd109a281a698c0f5ba31f158585b41f4f33659e54d3178443ab76a60e21690dbfb17f7f59f09ae3ea1647ec26ae49b14060660504f4da1c2059e1c5ab6810ac3d8e1258bd2f004a94ca0cd4c68fc1c061180610e96d645b12f47ae5cf4546b18538739e90f2edb0d8530e31a218e72b9480202acbaeb06178da78858e5e5c4705cdd4b668ffe3be5bae4867c9d5efe3a05efc62d60e1d19faeb56a80223cdd3472d791b7d32c05abb1cc00b6381fa0c4928f0c56fc14bc029b8809069093d712a3fd4dfab31963597e246ab29fc6ebedf2d392a51ab2dc5c59d0902a03132a84dfd920b35a3d0ba5f7a0635df298f9033e";
     const VAA_SET_FEE_1: vector<u8> =
        x"01000000000100181aa27fd44f3060fad0ae72895d42f97c45f7a5d34aa294102911370695e91e17ae82caa59f779edde2356d95cd46c2c381cdeba7a8165901a562374f212d750000bc614e000000000001000000000000000000000000000000000000000000000000000000000000000400000000000000010100000000000000000000000000000000000000000000000000000000436f7265030015000000000000000000000000000000000000000000000000000000000000015e";

    #[test]
    fun test_global_action() {
        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test setting sender to `caller`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_UPDATE_GUARDIAN_SET_1,
                &the_clock
            );
        let msg = governance_message::verify_vaa(&worm_state, verified_vaa);

        let expected_module = state::governance_module();
        let expected_action = 2;

        // Verify `GovernanceMessage` getters.
        assert!(governance_message::module_name(&msg) == expected_module, 0);
        assert!(governance_message::action(&msg) == expected_action, 0);
        assert!(governance_message::is_global_action(&msg), 0);
        assert!(!governance_message::is_local_action(&msg), 0);

        let expected_payload = governance_message::payload(&msg);

        // Take payload.
        let payload =
            governance_message::take_global_action(
                msg,
                expected_module,
                expected_action
            );
        assert!(payload == expected_payload, 0);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    fun test_local_action() {
        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test setting sender to `caller`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_SET_FEE_1,
                &the_clock
            );
        let msg = governance_message::verify_vaa(&worm_state, verified_vaa);

        let expected_module = state::governance_module();
        let expected_action = 3;

        // Verify `GovernanceMessage` getters.
        assert!(governance_message::module_name(&msg) == expected_module, 0);
        assert!(governance_message::action(&msg) == expected_action, 0);
        assert!(governance_message::is_local_action(&msg), 0);
        assert!(!governance_message::is_global_action(&msg), 0);

        let expected_payload = governance_message::payload(&msg);

        // Take payload.
        let payload =
            governance_message::take_local_action(
                msg,
                expected_module,
                expected_action
            );
        assert!(payload == expected_payload, 0);

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_INVALID_GOVERNANCE_MODULE
    )]
    fun test_cannot_assert_module_and_action_invalid_module() {
        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test setting sender to `caller`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_SET_FEE_1,
                &the_clock
            );
        let msg = governance_message::verify_vaa(&worm_state, verified_vaa);

        let expected_module = bytes32::default(); // all zeros
        let expected_action = 3;

        // Action agrees, but `assert_module_and_action` should fail.
        assert!(governance_message::action(&msg) == expected_action, 0);

        // You shall not pass!
        governance_message::assert_module_and_action_test_only(
            &msg,
            expected_module,
            expected_action
        );

        // Clean up.
        governance_message::destroy(msg);
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_INVALID_GOVERNANCE_ACTION
    )]
    fun test_cannot_assert_module_and_action_invalid_action() {
        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test setting sender to `caller`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_SET_FEE_1,
                &the_clock
            );
        let msg = governance_message::verify_vaa(&worm_state, verified_vaa);

        let expected_module = state::governance_module();
        let expected_action = 0;

        // Action agrees, but `assert_module_and_action` should fail.
        assert!(governance_message::module_name(&msg) == expected_module, 0);

        // You shall not pass!
        governance_message::assert_module_and_action_test_only(
            &msg,
            expected_module,
            expected_action
        );

        // Clean up.
        governance_message::destroy(msg);
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_GOVERNANCE_TARGET_CHAIN_NONZERO
    )]
    fun test_cannot_take_global_action_with_local() {
        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test setting sender to `caller`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_SET_FEE_1,
                &the_clock
            );
        let msg = governance_message::verify_vaa(&worm_state, verified_vaa);

        let expected_module = state::governance_module();
        let expected_action = 3;

        // Verify this message is not a global action.
        assert!(!governance_message::is_global_action(&msg), 0);

        // You shall not pass!
        governance_message::take_global_action(
            msg,
            expected_module,
            expected_action
        );

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_GOVERNANCE_TARGET_CHAIN_NOT_SUI
    )]
    fun test_cannot_take_local_action_with_invalid_chain() {
        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test setting sender to `caller`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_UPDATE_GUARDIAN_SET_1,
                &the_clock
            );
        let msg = governance_message::verify_vaa(&worm_state, verified_vaa);

        let expected_module = state::governance_module();
        let expected_action = 2;

        // Verify this message is not for Sui.
        assert!(!governance_message::is_local_action(&msg), 0);

        // You shall not pass!
        governance_message::take_local_action(
            msg,
            expected_module,
            expected_action
        );

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_OUTDATED_VERSION)]
    fun test_cannot_verify_vaa_outdated_version() {
        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        let wormhole_fee = 350;
        set_up_wormhole(scenario, wormhole_fee);

        // Prepare test setting sender to `caller`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        let verified_vaa =
            vaa::parse_and_verify(
                &worm_state,
                VAA_UPDATE_GUARDIAN_SET_1,
                &the_clock
            );

        state::migrate_version_test_only<V__0_1_0, V__MIGRATED>(
            &mut worm_state,
            version_control::first(),
            version_control::next_version()
        );

        // You shall not pass!
        let msg = governance_message::verify_vaa(&worm_state, verified_vaa);

        // Clean up.
        governance_message::destroy(msg);
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }
}
