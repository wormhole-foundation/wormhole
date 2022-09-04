module wormhole::vaa {
    use 0x1::vector;
    use 0x1::secp256k1::{Self};
    use 0x1::aptos_hash;

    use wormhole::u16::{U16};
    use wormhole::u32::{U32};
    use wormhole::deserialize;
    use wormhole::cursor::{Self};
    use wormhole::guardian_pubkey::{Self};
    use wormhole::structs::{
        Guardian,
        GuardianSet,
        Signature,
        create_signature,
        get_guardians,
        unpack_signature,
        get_address,
    };
    use wormhole::state;

    friend wormhole::guardian_set_upgrade;
    friend wormhole::contract_upgrade;

    const E_NO_QUORUM: u64 = 0x0;
    const E_TOO_MANY_SIGNATURES: u64 = 0x1;
    const E_INVALID_SIGNATURE: u64 = 0x2;
    const E_GUARDIAN_SET_EXPIRED: u64 = 0x3;
    const E_INVALID_GOVERNANCE_CHAIN: u64 = 0x4;
    const E_INVALID_GOVERNANCE_EMITTER: u64 = 0x5;
    const E_WRONG_VERSION: u64 = 0x6;
    const E_NON_INCREASING_SIGNERS: u64 = 0x7;

    struct VAA has key {
        // Header
        guardian_set_index: U32,
        signatures:         vector<Signature>,

        // Body
        timestamp:          U32,
        nonce:              U32,
        emitter_chain:      U16,
        emitter_address:    vector<u8>,
        sequence:           u64,
        consistency_level:  u8,
        hash:               vector<u8>, // 32 bytes
        payload:            vector<u8>, // variable bytes
    }

    //break

    fun parse(bytes: vector<u8>): VAA {
        let cur = cursor::init(bytes);
        let version = deserialize::deserialize_u8(&mut cur);
        assert!(version == 1, E_WRONG_VERSION);
        let guardian_set_index = deserialize::deserialize_u32(&mut cur);

        let signatures_len = deserialize::deserialize_u8(&mut cur);
        let signatures = vector::empty<Signature>();

        // TODO(csongor): I don't think we should assert this
        assert!(signatures_len <= 19, E_TOO_MANY_SIGNATURES);

        while ({
            spec {
                invariant signatures_len >  0;
                invariant signatures_len <= 19;
            };
            signatures_len > 0
        }) {
            let guardian_index = deserialize::deserialize_u8(&mut cur);
            let sig = deserialize::deserialize_vector(&mut cur, 64);
            let recovery_id = deserialize::deserialize_u8(&mut cur);
            let sig: secp256k1::ECDSASignature = secp256k1::ecdsa_signature_from_bytes(sig);
            vector::push_back(&mut signatures, create_signature(sig, recovery_id, guardian_index));
            signatures_len = signatures_len - 1;
        };

        let body = cursor::rest(cur);
        let hash = aptos_hash::keccak256(aptos_hash::keccak256(body));

        let cur = cursor::init(body);

        let timestamp = deserialize::deserialize_u32(&mut cur);
        let nonce = deserialize::deserialize_u32(&mut cur);
        let emitter_chain = deserialize::deserialize_u16(&mut cur);
        let emitter_address = deserialize::deserialize_vector(&mut cur, 32);
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

    public fun get_emitter_address(vaa: &VAA): vector<u8> {
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

    public fun verify(vaa: &VAA, guardian_set: &GuardianSet) {
        assert!(state::guardian_set_is_active(guardian_set), E_GUARDIAN_SET_EXPIRED);

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

            let address = guardian_pubkey::from_signature(hash, recovery_id, &sig);

            let cur_guardian = vector::borrow<Guardian>(&guardians, (guardian_index as u64));
            let cur_address = get_address(cur_guardian);

            assert!(address == cur_address, E_INVALID_SIGNATURE);

            sig_i = sig_i + 1;
        };
    }

    public fun parse_and_verify(bytes: vector<u8>): VAA {
        let vaa = parse(bytes);
        verify(&vaa, &state::get_current_guardian_set());
        vaa
    }

    /// Aborts if the VAA is not governance (i.e. sent from the governance
    /// emitter on the governance chain)
    public fun assert_governance(vaa: &VAA) {
        assert!(vaa.emitter_chain == state::get_governance_chain(), E_INVALID_GOVERNANCE_CHAIN);
        assert!(vaa.emitter_address == state::get_governance_contract(), E_INVALID_GOVERNANCE_EMITTER);
    }

    /// Aborts if the VAA has already been consumed. Marks the VAA as consumed
    /// the first time around.
    /// Only to be used for core bridge messages. Protocols should implement
    /// their own replay protection.
    public(friend) fun replay_protect(vaa: &VAA) {
        // this calls table::add which aborts if the key already exists
        state::set_governance_action_consumed(vaa.hash);
    }

    public fun quorum(num_guardians: u64): u64 {
        (num_guardians * 2) / 3 + 1
    }

}
