module wormhole::vaa {
    use 0x1::vector;
    use 0x1::secp256k1::{Self};
    use 0x1::hash::{Self};
    // use 0x1::timestamp::{Self};

    use wormhole::u16::{U16};
    use wormhole::u32::{U32};
    use wormhole::deserialize;
    use wormhole::cursor::{Self};
    use wormhole::guardian_pubkey::{Self};
    use wormhole::serialize;
    use wormhole::structs::{
        Guardian,
        GuardianSet,
        Signature,
        create_signature,
        get_guardians,
        unpack_signature,
        get_address,
    };
    use wormhole::state::{get_current_guardian_set};

    const E_NO_QUORUM: u64 = 0x0;
    const E_TOO_MANY_SIGNATURES: u64 = 0x1;
    const E_INVALID_SIGNATURE: u64 = 0x2;

    struct VAA has key {
        // Header
        version:            u8,
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

    public fun parse(bytes: vector<u8>): VAA {

        let cur = cursor::init(bytes);
        let version = deserialize::deserialize_u8(&mut cur);
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
        let hash = hash::sha3_256(hash::sha3_256(body));

        let cur = cursor::init(body);

        let timestamp = deserialize::deserialize_u32(&mut cur);
        let nonce = deserialize::deserialize_u32(&mut cur);
        let emitter_chain = deserialize::deserialize_u16(&mut cur);
        let emitter_address = deserialize::deserialize_vector(&mut cur, 32);
        let sequence = deserialize::deserialize_u64(&mut cur);
        let consistency_level = deserialize::deserialize_u8(&mut cur);

        let payload = cursor::rest(cur);

        VAA {
            version,
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

    public fun get_version(vaa: &VAA): u8 {
         vaa.version
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
            version: _,
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

    public fun verify(vaa: &VAA, guardian_set: GuardianSet) {
        let guardians = get_guardians(guardian_set);
        let hash = vaa.hash;
        let n = vector::length<Signature>(&vaa.signatures);
        let m = vector::length<Guardian>(&guardians);

        assert!(n >= quorum(m), E_NO_QUORUM);

        // TODO: check expiration time once comparison operation is implemented for U32 type
        //if (get_guardian_set_index(guardian_set) != get_current_guardian_set_index() && get_guardian_set_expiry(guardianSet) < timestamp::now_seconds()){
        //    return (false, string::utf8(b"Guardian set expired"))
        //};

        // TODO(csongor): check that the guardian indices are strictly increasing
        let i = 0;
        while (i < n) {
            let (sig, recovery_id, guardian_index) = unpack_signature(vector::borrow(&vaa.signatures, i));
            let address = guardian_pubkey::from_signature(hash, recovery_id, &sig);

            let cur_guardian = vector::borrow<Guardian>(&guardians, (guardian_index as u64));
            let cur_address = get_address(*cur_guardian);

            assert!(address == cur_address, E_INVALID_SIGNATURE);

            i = i + 1;
        };
    }

    public entry fun parse_and_verify(bytes: vector<u8>): VAA {
        let vaa = parse(bytes);
        verify(&vaa, get_current_guardian_set());
        vaa
    }

    //TODO: we shouldn't reserialise the VAA to copmute its hash. However, this
    // functions might be useful in testing
    fun hash(vaa: &VAA): vector<u8> {
        let bytes = vector::empty<u8>();
        serialize::serialize_u32(&mut bytes, vaa.timestamp);
        serialize::serialize_u32(&mut bytes, vaa.nonce);
        serialize::serialize_u16(&mut bytes, vaa.emitter_chain);
        serialize::serialize_vector(&mut bytes, vaa.emitter_address);
        serialize::serialize_u64(&mut bytes, vaa.sequence);
        serialize::serialize_u8(&mut bytes, vaa.consistency_level);
        serialize::serialize_vector(&mut bytes, vaa.payload);
        hash::sha3_256(bytes)
    }

    public fun quorum(num_guardians: u64): u64 {
        (num_guardians * 2) / 3 + 1
    }

}
