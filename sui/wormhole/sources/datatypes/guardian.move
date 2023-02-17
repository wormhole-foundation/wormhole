module wormhole::guardian {
    use std::vector::{Self};
    use sui::ecdsa_k1::{Self};

    use wormhole::bytes20::{Self, Bytes20};
    use wormhole::guardian_signature::{Self, GuardianSignature};

    const E_INVALID_EC_PUBKEY_LENGTH: u64 = 0;

    const PUBKEY_LENGTH: u64 = 20;

    struct Guardian has store, drop, copy {
        pubkey: Bytes20
    }

    public fun new(pubkey: vector<u8>): Guardian {
        Guardian { pubkey: bytes20::new(pubkey) }
    }

    public fun pubkey(self: &Guardian): Bytes20 {
        self.pubkey
    }

    public fun as_bytes(self: &Guardian): vector<u8> {
        bytes20::data(&self.pubkey)
    }

    public fun to_bytes(value: Guardian): vector<u8> {
        bytes20::to_bytes(value.pubkey)
    }

    public fun verify(
        self: &Guardian,
        signature: GuardianSignature,
        message_hash: vector<u8>
    ): bool {
        let (rs, recovery_id, _) = guardian_signature::destroy(signature);
        as_bytes(self) == ecrecover(message_hash, recovery_id, rs)
    }

    /// Same as 'ecrecover' in EVM.
    fun ecrecover(
        message: vector<u8>,
        recovery_id: u8,
        sig: vector<u8>,
    ): vector<u8> {
        // sui's ecrecover function takes a 65 byte array (signature + recovery byte)
        vector::push_back(&mut sig, recovery_id);

        let pubkey =
            ecdsa_k1::decompress_pubkey(&ecdsa_k1::ecrecover(&sig, &message));

        // decompress_pubkey returns 65 bytes, the first byte is not relevant to
        // us, so we remove it
        vector::remove(&mut pubkey, 0);

        let hash = ecdsa_k1::keccak256(&pubkey);
        let guardian_pubkey = vector::empty<u8>();
        let i = 0;
        while (i < PUBKEY_LENGTH) {
            vector::push_back(
                &mut guardian_pubkey,
                vector::pop_back(&mut hash)
            );
            i = i + 1;
        };
        vector::reverse(&mut guardian_pubkey);

        guardian_pubkey
    }
}
