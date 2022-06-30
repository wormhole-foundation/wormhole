module Wormhole::VAA {
    use Sui::ID::VersionedID;
    use Wormhole::Parse;
    use Wormhole::Serialize;
    use Wormhole::GuardianSet;

    // Note: VAA fields are all u64 due to move missing u16/u32 types, see VAA spec for real
    // field sizes.
    struct VAA has key, drop {
        id:                 VersionedID,

        // Header
        version:            u8,
        guardian_set_index: u64,
        signatures:         vector<vector<u8>>,

        // Body
        timestamp:          u64,
        nonce:              u64,
        emitter_chain:      u64,
        emitter_address:    vector<u8>,
        sequence:           u64,
        consistency_level:  u8,
        payload:            vector<u8>,
    }

    public fun parse(bytes: vector<u8>): VAA {
        use Sui::TxContext;

        let (version, bytes) = Parse::parse_u8(bytes);
        let (guardian_set_index, bytes) = Parse::parse_u32(bytes);

        let (signatures_len, bytes) = Parse::parse_u8(bytes);
        let signatures = Vector::empty();

        assert!(signatures_len <= 19, 0);

        while ({
            spec { 
                invariant signatures_len >  0;
                invariant signatures_len <= 19;
            };
            signatures_len > 0
        }) {
            let (signature, r) = Parse::parse_vector(bytes, 32);
            Vector::push_back(&mut signatures, signature);
            signatures_len = signatures_len - 1;
            bytes = r;
        };

        let (timestamp, bytes) = Parse::parse_u32(bytes);
        let (nonce, bytes) = Parse::parse_u32(bytes);
        let (emitter_chain, bytes) = Parse::parse_u16(bytes);
        let (emitter_address, bytes) = Parse::parse_vector(bytes, 20);
        let (sequence, bytes) = Parse::parse_u64(bytes);
        let (consistency_level, bytes) = Parse::parse_u8(bytes);
        let remaining_length = Vector::length(&bytes);
        let (payload, _) = Parse::parse_vector(bytes, remaining_length);

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
            payload:            payload,
        }
    }

    public(script) fun verify(vaa: &VAA, guardian_set: &GuardianSet::GuardianSet) {
        use Sui::Signer::secp256k_ecrecover;
        use Sui::Signer::secp256k_verify;
        let index = 0;
        let hash = hash(&vaa);
        for signature in vaa.signatures {
            let pubkey = secp256k_ecrecover(&signature);
            assert!(expected_signers[index] == pubkey, 0);
            secp256k_verify(&signature, &pubkey, &hash);
            index = index + 1;
        }
    }

    fun hash(vaa: &VAA): vector<u8> {
        use Sui::Hash;

        let mut bytes = Vector::empty();
        Serialize::serialize_u32(&mut bytes, vaa.timestamp);
        Serialize::serialize_u32(&mut bytes, vaa.nonce);
        Serialize::serialize_u16(&mut bytes, vaa.emitter_chain);
        Serialize::serialize_vector(&mut bytes, vaa.emitter_address);
        Serialize::serialize_u64(&mut bytes, vaa.sequence);
        Serialize::serialize_u8(&mut bytes, vaa.consistency_level);
        Serialize::serialize_vector(&mut bytes, vaa.payload);

        Hash::sha3_256(bytes)
    }
}
