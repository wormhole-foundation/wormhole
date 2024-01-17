// SPDX-License-Identifier: Apache 2

/// This module implements a custom type representing a Guardian governance
/// action. Each governance action has an associated module name, relevant chain
/// and payload encoding instructions/data used to perform an administrative
/// change on a contract.
module wormhole::governance_message {
    use wormhole::bytes::{Self};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::consumed_vaas::{Self, ConsumedVAAs};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{ExternalAddress};
    use wormhole::state::{Self, State, chain_id};
    use wormhole::vaa::{Self, VAA};

    /// Guardian set used to sign VAA did not use current Guardian set.
    const E_OLD_GUARDIAN_SET_GOVERNANCE: u64 = 0;
    /// Governance chain does not match.
    const E_INVALID_GOVERNANCE_CHAIN: u64 = 1;
    /// Governance emitter address does not match.
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

    /// The public constructors for `DecreeTicket` (`authorize_verify_global`
    /// and `authorize_verify_local`) require a witness of type `T`. This is to
    /// ensure that `DecreeTicket`s cannot be mixed up between modules
    /// maliciously.
    struct DecreeTicket<phantom T> {
        governance_chain: u16,
        governance_contract: ExternalAddress,
        module_name: Bytes32,
        action: u8,
        global: bool
    }

    struct DecreeReceipt<phantom T> {
        payload: vector<u8>,
        digest: Bytes32,
        sequence: u64
    }

    /// This method prepares `DecreeTicket` for global governance action. This
    /// means the VAA encodes target chain ID == 0.
    public fun authorize_verify_global<T: drop>(
        _witness: T,
        governance_chain: u16,
        governance_contract: ExternalAddress,
        module_name: Bytes32,
        action: u8
    ): DecreeTicket<T> {
        DecreeTicket {
            governance_chain,
            governance_contract,
            module_name,
            action,
            global: true
        }
    }

    /// This method prepares `DecreeTicket` for local governance action. This
    /// means the VAA encodes target chain ID == 21 (Sui's).
    public fun authorize_verify_local<T: drop>(
        _witness: T,
        governance_chain: u16,
        governance_contract: ExternalAddress,
        module_name: Bytes32,
        action: u8
    ): DecreeTicket<T> {
        DecreeTicket {
            governance_chain,
            governance_contract,
            module_name,
            action,
            global: false
        }
    }

    public fun sequence<T>(receipt: &DecreeReceipt<T>): u64 {
        receipt.sequence
    }

    /// This method unpacks `DecreeReceipt` and puts the VAA digest into a
    /// `ConsumedVAAs` container. Then it returns the governance payload.
    public fun take_payload<T>(
        consumed: &mut ConsumedVAAs,
        receipt: DecreeReceipt<T>
    ): vector<u8> {
        let DecreeReceipt { payload, digest, sequence: _ } = receipt;

        consumed_vaas::consume(consumed, digest);

        payload
    }

    /// Method to peek into the payload in `DecreeReceipt`.
    public fun payload<T>(receipt: &DecreeReceipt<T>): vector<u8> {
        receipt.payload
    }

    /// Destroy the receipt.
    public fun destroy<T>(receipt: DecreeReceipt<T>) {
        let DecreeReceipt { payload: _, digest: _, sequence: _ } = receipt;
    }

    /// This method unpacks a `DecreeTicket` to validate its members to make
    /// sure that the parameters match what was encoded in the VAA.
    public fun verify_vaa<T>(
        wormhole_state: &State,
        verified_vaa: VAA,
        ticket: DecreeTicket<T>
    ): DecreeReceipt<T> {
        state::assert_latest_only(wormhole_state);

        let DecreeTicket {
            governance_chain,
            governance_contract,
            module_name,
            action,
            global
        } = ticket;

        // Protect against governance actions enacted using an old guardian set.
        // This is not a protection found in the other Wormhole contracts.
        assert!(
            vaa::guardian_set_index(&verified_vaa) == state::guardian_set_index(wormhole_state),
            E_OLD_GUARDIAN_SET_GOVERNANCE
        );

        // Both the emitter chain and address must equal.
        assert!(
            vaa::emitter_chain(&verified_vaa) == governance_chain,
            E_INVALID_GOVERNANCE_CHAIN
        );
        assert!(
            vaa::emitter_address(&verified_vaa) == governance_contract,
            E_INVALID_GOVERNANCE_EMITTER
        );

        // Cache VAA digest.
        let digest = vaa::digest(&verified_vaa);

        // Get the VAA sequence number.
        let sequence = vaa::sequence(&verified_vaa);

        // Finally deserialize Wormhole payload as governance message.
        let (
            parsed_module_name,
            parsed_action,
            chain,
            payload
        ) = deserialize(vaa::take_payload(verified_vaa));

        assert!(module_name == parsed_module_name, E_INVALID_GOVERNANCE_MODULE);
        assert!(action == parsed_action, E_INVALID_GOVERNANCE_ACTION);

        // Target chain, which determines whether the governance VAA applies to
        // all chains or Sui.
        if (global) {
            assert!(chain == 0, E_GOVERNANCE_TARGET_CHAIN_NONZERO);
        } else {
            assert!(chain == chain_id(), E_GOVERNANCE_TARGET_CHAIN_NOT_SUI);
        };

        DecreeReceipt { payload, digest, sequence }
    }

    fun deserialize(buf: vector<u8>): (Bytes32, u8, u16, vector<u8>) {
        let cur = cursor::new(buf);

        (
            bytes32::take_bytes(&mut cur),
            bytes::take_u8(&mut cur),
            bytes::take_u16_be(&mut cur),
            cursor::take_rest(cur)
        )
    }

    #[test_only]
    public fun deserialize_test_only(
        buf: vector<u8>
    ): (
        Bytes32,
        u8,
        u16,
        vector<u8>
    ) {
        deserialize(buf)
    }

    #[test_only]
    public fun take_decree(buf: vector<u8>): vector<u8> {
        let (_, _, _, payload) = deserialize(buf);
        payload
    }
}

#[test_only]
module wormhole::governance_message_tests {
    use sui::test_scenario::{Self};
    use sui::tx_context::{Self};

    use wormhole::bytes32::{Self};
    use wormhole::consumed_vaas::{Self};
    use wormhole::external_address::{Self};
    use wormhole::governance_message::{Self};
    use wormhole::state::{Self};
    use wormhole::vaa::{Self};
    use wormhole::version_control::{Self};
    use wormhole::wormhole_scenario::{
        set_up_wormhole,
        person,
        return_clock,
        return_state,
        take_clock,
        take_state
    };

    struct GovernanceWitness has drop {}

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
        let (
            _,
            _,
            _,
            expected_payload
        ) = governance_message::deserialize_test_only(
            vaa::payload(&verified_vaa)
        );

        let ticket =
            governance_message::authorize_verify_global(
                GovernanceWitness {},
                state::governance_chain(&worm_state),
                state::governance_contract(&worm_state),
                state::governance_module(),
                2 // update guadian set
            );
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);

        let consumed = consumed_vaas::new(&mut tx_context::dummy());
        let payload = governance_message::take_payload(&mut consumed, receipt);
        assert!(payload == expected_payload, 0);

        // Clean up.
        consumed_vaas::destroy(consumed);
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
            vaa::parse_and_verify(&worm_state, VAA_SET_FEE_1, &the_clock);
        let (
            _,
            _,
            _,
            expected_payload
        ) = governance_message::deserialize_test_only(
            vaa::payload(&verified_vaa)
        );

        let ticket =
            governance_message::authorize_verify_local(
                GovernanceWitness {},
                state::governance_chain(&worm_state),
                state::governance_contract(&worm_state),
                state::governance_module(),
                3 // set fee
            );
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);

        let consumed = consumed_vaas::new(&mut tx_context::dummy());
        let payload = governance_message::take_payload(&mut consumed, receipt);
        assert!(payload == expected_payload, 0);

        // Clean up.
        consumed_vaas::destroy(consumed);
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_INVALID_GOVERNANCE_CHAIN
    )]
    fun test_cannot_verify_vaa_invalid_governance_chain() {
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
            vaa::parse_and_verify(&worm_state, VAA_SET_FEE_1, &the_clock);

        // Show that this emitter chain ID does not equal the encoded one.
        let invalid_chain = 0xffff;
        assert!(invalid_chain != vaa::emitter_chain(&verified_vaa), 0);

        let ticket =
            governance_message::authorize_verify_local(
                GovernanceWitness {},
                invalid_chain,
                state::governance_contract(&worm_state),
                state::governance_module(),
                3 // set fee
            );

        // You shall not pass!
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);

        // Clean up.
        governance_message::destroy(receipt);

        abort 42
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_INVALID_GOVERNANCE_EMITTER
    )]
    fun test_cannot_verify_vaa_invalid_governance_emitter() {
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
            vaa::parse_and_verify(&worm_state, VAA_SET_FEE_1, &the_clock);

        // Show that this emitter address does not equal the encoded one.
        let invalid_emitter = external_address::new(bytes32::default());
        assert!(invalid_emitter != vaa::emitter_address(&verified_vaa), 0);

        let ticket =
            governance_message::authorize_verify_global(
                GovernanceWitness {},
                state::governance_chain(&worm_state),
                invalid_emitter,
                state::governance_module(),
                3 // set fee
            );

        // You shall not pass!
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);

        // Clean up.
        governance_message::destroy(receipt);

        abort 42
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_INVALID_GOVERNANCE_MODULE
    )]
    fun test_cannot_verify_vaa_invalid_governance_module() {
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
            vaa::parse_and_verify(&worm_state, VAA_SET_FEE_1, &the_clock);
        let (
            expected_module,
            _,
            _,
            _
        ) = governance_message::deserialize_test_only(
            vaa::payload(&verified_vaa)
        );

        // Show that this module does not equal the encoded one.
        let invalid_module = bytes32::from_bytes(b"Not Wormhole");
        assert!(invalid_module != expected_module, 0);

        let ticket =
            governance_message::authorize_verify_local(
                GovernanceWitness {},
                state::governance_chain(&worm_state),
                state::governance_contract(&worm_state),
                invalid_module,
                3 // set fee
            );

        // You shall not pass!
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);

        // Clean up.
        governance_message::destroy(receipt);

        abort 42
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_INVALID_GOVERNANCE_ACTION
    )]
    fun test_cannot_verify_vaa_invalid_governance_action() {
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
            vaa::parse_and_verify(&worm_state, VAA_SET_FEE_1, &the_clock);
        let (
            _,
            expected_action,
            _,
            _
        ) = governance_message::deserialize_test_only(
            vaa::payload(&verified_vaa)
        );

        // Show that this action does not equal the encoded one.
        let invalid_action = 0xff;
        assert!(invalid_action != expected_action, 0);

        let ticket =
            governance_message::authorize_verify_local(
                GovernanceWitness {},
                state::governance_chain(&worm_state),
                state::governance_contract(&worm_state),
                state::governance_module(),
                invalid_action
            );

        // You shall not pass!
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);

        // Clean up.
        governance_message::destroy(receipt);

        abort 42
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_GOVERNANCE_TARGET_CHAIN_NONZERO
    )]
    fun test_cannot_verify_vaa_governance_target_chain_nonzero() {
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
            vaa::parse_and_verify(&worm_state, VAA_SET_FEE_1, &the_clock);
        let (
            _,
            _,
            expected_target_chain,
            _
        ) = governance_message::deserialize_test_only(
            vaa::payload(&verified_vaa)
        );

        // Show that this target chain ID does reflect a global action.
        let not_global = expected_target_chain != 0;
        assert!(not_global, 0);

        let ticket =
            governance_message::authorize_verify_global(
                GovernanceWitness {},
                state::governance_chain(&worm_state),
                state::governance_contract(&worm_state),
                state::governance_module(),
                3 // set fee
            );

        // You shall not pass!
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);

        // Clean up.
        governance_message::destroy(receipt);

        abort 42
    }

    #[test]
    #[expected_failure(
        abort_code = governance_message::E_GOVERNANCE_TARGET_CHAIN_NOT_SUI
    )]
    fun test_cannot_verify_vaa_governance_target_chain_not_sui() {
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
        let (
            _,
            _,
            expected_target_chain,
            _
        ) = governance_message::deserialize_test_only(
            vaa::payload(&verified_vaa)
        );

        // Show that this target chain ID does reflect a global action.
        let global = expected_target_chain == 0;
        assert!(global, 0);

        let ticket =
            governance_message::authorize_verify_local(
                GovernanceWitness {},
                state::governance_chain(&worm_state),
                state::governance_contract(&worm_state),
                state::governance_module(),
                2 // update guardian set
            );

        // You shall not pass!
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);

        // Clean up.
        governance_message::destroy(receipt);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_NOT_CURRENT_VERSION)]
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
            vaa::parse_and_verify(&worm_state, VAA_SET_FEE_1, &the_clock);
        let ticket =
            governance_message::authorize_verify_local(
                GovernanceWitness {},
                state::governance_chain(&worm_state),
                state::governance_contract(&worm_state),
                state::governance_module(),
                3 // set fee
            );

        // Conveniently roll version back.
        state::reverse_migrate_version(&mut worm_state);

        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        state::migrate_version_test_only(
            &mut worm_state,
            version_control::previous_version_test_only(),
            version_control::next_version()
        );

        // You shall not pass!
        let receipt =
            governance_message::verify_vaa(&worm_state, verified_vaa, ticket);

        // Clean up.
        governance_message::destroy(receipt);

        abort 42
    }
}
