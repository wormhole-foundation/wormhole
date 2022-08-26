module Wormhole::VAA{
    use 0x1::vector;
    use 0x1::secp256k1::{Self};
    use 0x1::hash::{Self};
    // use 0x1::timestamp::{Self};

    use Wormhole::Uints::{U16, U32};
    use Wormhole::Deserialize;
    use Wormhole::cursor::{Self};
    use Wormhole::guardian_pubkey::{Self};
    use Wormhole::Serialize;
    use Wormhole::Structs::{GuardianSet, Guardian, getGuardians, Signature, unpackSignature, createSignature};
    use Wormhole::State::{getCurrentGuardianSet};

    const E_NO_QUORUM: u64 = 0x0;
    const E_TOO_MANY_SIGNATURES: u64 = 0x1;

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
        let version = Deserialize::deserialize_u8(&mut cur);
        let guardian_set_index = Deserialize::deserialize_u32(&mut cur);

        let signatures_len = Deserialize::deserialize_u8(&mut cur);
        let signatures = vector::empty<Signature>();

        assert!(signatures_len <= 19, E_TOO_MANY_SIGNATURES);

         while ({
            spec {
                invariant signatures_len >  0;
                invariant signatures_len <= 19;
            };
            signatures_len > 0
        }) {
            let sig = Deserialize::deserialize_vector(&mut cur, 64);
            let recovery_id = Deserialize::deserialize_u8(&mut cur);
            let sig: secp256k1::ECDSASignature = secp256k1::ecdsa_signature_from_bytes(sig);
            let guardianIndex = Deserialize::deserialize_u32(&mut cur);
            vector::push_back(&mut signatures, createSignature(sig, recovery_id, guardianIndex));
            signatures_len = signatures_len - 1;
        };

        let timestamp = Deserialize::deserialize_u32(&mut cur);
        let nonce = Deserialize::deserialize_u32(&mut cur);
        let emitter_chain = Deserialize::deserialize_u16(&mut cur);
        let emitter_address = Deserialize::deserialize_vector(&mut cur, 32);
        let sequence = Deserialize::deserialize_u64(&mut cur);
        let consistency_level = Deserialize::deserialize_u8(&mut cur);
        let hash = Deserialize::deserialize_vector(&mut cur, 32);

        let remaining_length = vector::length(&bytes);
        let payload = Deserialize::deserialize_vector(&mut cur, remaining_length);

        cursor::destroy_empty(cur);

        VAA {
            version:            version,
            guardian_set_index: guardian_set_index,
            signatures:         signatures,
            timestamp:          timestamp,
            nonce:              nonce,
            emitter_chain:      emitter_chain,
            emitter_address:    emitter_address,
            sequence:           sequence,
            consistency_level:  consistency_level,
            hash:               hash,
            payload:            payload,
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

    //TODO: why does this return the payload?
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
         //(id, version, guardian_set_index, signatures, timestamp, nonce, emitter_chain, emitter_address, sequence, consistency_level, payload)
        payload
    }

    public fun verify(vaa: &VAA, guardianSet: GuardianSet) {
        let guardians = getGuardians(guardianSet);
        let hash = hash(vaa);
        let n = vector::length<Signature>(&vaa.signatures);
        let m = vector::length<Guardian>(&guardians);

        assert!(n >= quorum(m), E_NO_QUORUM);

        // TODO: check expiration time once comparison operation is implemented for U32 type
        //if (getGuardianSetIndex(guardianSet) != getCurrentGuardianSetIndex() && getGuardianSetExpiry(guardianSet) < timestamp::now_seconds()){
        //    return (false, string::utf8(b"Guardian set expired"))
        //};

        let i = 0;
        while (i < n) {
            let (sig, recovery_id, _) = unpackSignature(vector::borrow(&vaa.signatures, i));
            let _address = guardian_pubkey::from_signature(hash, recovery_id, &sig);

            // TODO - index into guardians and check pubkey
            // let cur_guardian = vector::borrow<Guardian>(&guardians, guardianSetIndex);
            // let cur_address = getAddress(*cur_guardian);

            // if (cur_address != address) {
            //    return (false, string::utf8(b"Invalid signature"))
            // };

            i = i + 1;
        };
    }

    public entry fun parse_and_verify(bytes: vector<u8>): VAA {
        let vaa = parse(bytes);
        verify(&vaa, getCurrentGuardianSet());
        vaa
    }

    //TODO: we shouldn't reserialise the VAA to copmute its hash. However, this
    // functions might be useful in testing
    fun hash(vaa: &VAA): vector<u8> {
        let bytes = vector::empty<u8>();
        Serialize::serialize_u32(&mut bytes, vaa.timestamp);
        Serialize::serialize_u32(&mut bytes, vaa.nonce);
        Serialize::serialize_u16(&mut bytes, vaa.emitter_chain);
        Serialize::serialize_vector(&mut bytes, vaa.emitter_address);
        Serialize::serialize_u64(&mut bytes, vaa.sequence);
        Serialize::serialize_u8(&mut bytes, vaa.consistency_level);
        Serialize::serialize_vector(&mut bytes, vaa.payload);
        hash::sha3_256(bytes)
    }

    public fun quorum(numGuardians: u64): u64 {
        (numGuardians * 2) / 3 + 1
    }

}
