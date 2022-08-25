module Wormhole::VAA{
    use 0x1::vector;
    use 0x1::string::{Self, String};
    use 0x1::secp256k1::{Self};
    use 0x1::hash::{Self};
    use 0x1::timestamp::{Self};
    use Wormhole::Deserialize;
    use Wormhole::Serialize;
    use Wormhole::Structs::{GuardianSet, Guardian, getAddress, getGuardians, getGuardianSetIndex, getGuardianSetExpiry, Signature, unpackSignature, createSignature};
    use Wormhole::State::{getCurrentGuardianSet, getCurrentGuardianSetIndex};

    struct VAA has key {
            // Header
            version:            u8,
            guardian_set_index: u64,
            signatures:         vector<Signature>,

            // Body
            timestamp:          u64,
            nonce:              u64,
            emitter_chain:      u64,
            emitter_address:    vector<u8>,
            sequence:           u64,
            consistency_level:  u8,
            hash:               vector<u8>,
            payload:            vector<u8>,
    }

    //break

    public fun parse(bytes: vector<u8>): VAA {
        let (version, bytes) = Deserialize::deserialize_u8(bytes);
        let (guardian_set_index, bytes) = Deserialize::deserialize_u64(bytes);

        let (signatures_len, bytes) = Deserialize::deserialize_u8(bytes);
        let signatures = vector::empty<Signature>();

        assert!(signatures_len <= 19, 0);

         while ({
            spec {
                invariant signatures_len >  0;
                invariant signatures_len <= 19;
            };
            signatures_len > 0
        }) {
            let (signature, _) = Deserialize::deserialize_vector(bytes, 32);
            let (guardianIndex, _) = Deserialize::deserialize_u64(bytes);
            vector::push_back(&mut signatures, createSignature(signature, guardianIndex));
            signatures_len = signatures_len - 1;
        };

        let (timestamp, bytes) = Deserialize::deserialize_u64(bytes);
        let (nonce, bytes) = Deserialize::deserialize_u64(bytes);
        let (emitter_chain, bytes) = Deserialize::deserialize_u64(bytes);
        let (emitter_address, bytes) = Deserialize::deserialize_vector(bytes, 20);
        let (sequence, bytes) = Deserialize::deserialize_u64(bytes);
        let (consistency_level, bytes) = Deserialize::deserialize_u8(bytes);
        let (hash, bytes) = Deserialize::deserialize_vector(bytes, 32);

        let remaining_length = vector::length(&bytes);
        let (payload, _) = Deserialize::deserialize_vector(bytes, remaining_length);

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

    public fun get_version(vaa: &VAA): u8{
         vaa.version
    }

    public fun get_guardian_set_index(vaa: &VAA): u64{
         vaa.guardian_set_index
    }

    public fun get_timestamp(vaa: &VAA): u64{
         vaa.timestamp
    }

    public fun get_payload(vaa: &VAA): vector<u8>{
         vaa.payload
    }

    public fun get_hash(vaa: &VAA): vector<u8>{
         vaa.hash
    }

    public fun get_emitter_chain(vaa: &VAA): u64{
         vaa.emitter_chain
    }

    public fun get_emitter_address(vaa: &VAA): vector<u8>{
         vaa.emitter_address
    }

    public fun get_sequence(vaa: &VAA): u64{
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

    public fun verifyVAA(vaa: &VAA, guardianSet: GuardianSet): (bool, String) {//, guardian_set: &GuardianSet::GuardianSet) {
        let guardians = getGuardians(guardianSet);
        let hash = hash(vaa);
        let n = vector::length<Signature>(&vaa.signatures);
        let m = vector::length<Guardian>(&guardians);

        if (n < quorum(m)){
            return (false, string::utf8(b"Quorum not met"))
        };

        if (getGuardianSetIndex(guardianSet) != getCurrentGuardianSetIndex() && getGuardianSetExpiry(guardianSet) < timestamp::now_seconds()){
            return (false, string::utf8(b"Guardian set expired"))
        };

        let i = 0;
        while (i < n) {
            let (sig, guardianSetIndex) = unpackSignature(vector::borrow(&vaa.signatures, i));
            let sig: secp256k1::ECDSASignature = secp256k1::ecdsa_signature_from_bytes(sig);

            let pubkey = secp256k1::ecdsa_recover(hash, 0, &sig);
            let pubkey = std::option::extract(&mut pubkey);
            let address = addresFromPubkey(&pubkey);

            let cur_guardian = vector::borrow<Guardian>(&guardians, guardianSetIndex);
            let cur_address = getAddress(*cur_guardian);

            if (cur_address != address) {
               return (false, string::utf8(b"Invalid signature"))
            };

            i = i + 1;
        };
        (true, string::utf8(b""))
    }

    public entry fun parseAndVerifyVAA(encodedVM: vector<u8>): (VAA, bool, String) {
        let vaa = parse(encodedVM);
        let (valid, reason) = verifyVAA(&vaa, getCurrentGuardianSet());
        (vaa, valid, reason)
    }

    /// Converts a 64 byte secpk256k1 public key into an EVM-style 20 byte address.
    ///
    /// The address is derived by taking the last 20 bytes of the keccak256 hash
    /// of the public key.
    /// TODO: add tests for this
    fun addresFromPubkey(pubkey: &secp256k1::ECDSARawPublicKey): vector<u8> {
        let bytes = secp256k1::ecdsa_raw_public_key_to_bytes(pubkey);
        let hash = hash::sha3_256(bytes);
        let address = vector::empty<u8>();
        let i = 0;
        // the hash is 32 bytes, but can't hurt to compute it
        let len = vector::length(&hash);
        let start = len - 20 - 1;
        while (i < 20) {
            vector::push_back(&mut address, *vector::borrow(&hash, start + i));
            i = i + 1;
        };
        address
    }

    //TODO: we shouldn't reserialise the VAA to copmute its hash. However, this
    // functions might be useful in testing
    fun hash(vaa: &VAA): vector<u8> {
        let bytes = vector::empty<u8>();
        Serialize::serialize_u64(&mut bytes, vaa.timestamp);
        Serialize::serialize_u64(&mut bytes, vaa.nonce);
        Serialize::serialize_u64(&mut bytes, vaa.emitter_chain);
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
