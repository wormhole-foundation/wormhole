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
    use sui::test_scenario::{Self, Scenario, next_tx, ctx, take_owned, return_owned};

    fun scenario(): Scenario { test_scenario::begin(&@0x123233) }
    fun people(): (address, address, address) { (@0x124323, @0xE05, @0xFACE) }

    use wormhole::guardian_set_upgrade::{do_upgrade_test};
    use wormhole::state::{Self, State, test_init};
    use wormhole::structs::{Self};
    use wormhole::myu32::{Self as u32};
    //use wormhole::myvaa::{Self as vaa};

    ///// A test VAA signed by the first guardian set (index 0) containing guardian a single
    ///// guardian beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe
    ///// It's a governance VAA (contract upgrade), so we can test all sorts of
    ///// properties
    // const GOV_VAA: vector<u8> = x"010000000001000da16466429ee8ffb09b90ca90db8326d20cfeeae0542da9dcaaad641a5aca2d6c1fe33a5970ca84fd0ff5e6d29ef9e40404eb1a8892b509f085fc725b9e23a30100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000020b10360000000000000000000000000000000000000000000000000000000000436f7265010016d8f30e4a345ea0fa5df11daac4e1866ee368d253209cf9eda012d915a2db09e6";

    #[test]
    fun test_upgrade_guardian() {
        test_upgrade_guardian_(&mut scenario())
    }

    fun test_upgrade_guardian_(test: &mut Scenario) {
        let (admin, _, _) = people();
        next_tx(test, &admin); {
            test_init(ctx(test));
        };
        next_tx(test, &admin);{
            let state = take_owned<State>(test);
            // first store a guardian set within State at index 0
            let initial_guardian = vector[structs::create_guardian(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe")];
            state::store_guardian_set(&mut state, u32::from_u64(0), structs::create_guardian_set(u32::from_u64(0), initial_guardian));
            let new_guardians = vector[structs::create_guardian(x"71aa1be1d36cafe3867910f99c09e347899c19c3")];

            // upgrade guardian set
            do_upgrade_test(&mut state, u32::from_u64(1), new_guardians, ctx(test));
            assert!(state::get_current_guardian_set_index(&state)==u32::from_u64(1), 0);

            // return state
            return_owned(test, state);
        }
    }

}
