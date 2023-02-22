module wormhole::guardian {
    use std::vector::{Self};
    use sui::ecdsa_k1::{Self};

    use wormhole::bytes20::{Self, Bytes20};
    use wormhole::guardian_signature::{Self, GuardianSignature};

    const E_INVALID_EC_PUBKEY_LENGTH: u64 = 0;
    const E_ZERO_ADDRESS: u64 = 1;

    const PUBKEY_LENGTH: u64 = 20;

    struct Guardian has store {
        pubkey: Bytes20
    }

    public fun new(pubkey: vector<u8>): Guardian {
        let data = bytes20::new(pubkey);
        assert!(bytes20::is_nonzero(&data), E_ZERO_ADDRESS);
        Guardian { pubkey: data }
    }

    public fun pubkey(self: &Guardian): Bytes20 {
        self.pubkey
    }

    public fun as_bytes(self: &Guardian): vector<u8> {
        bytes20::data(&self.pubkey)
    }

    public fun verify(
        self: &Guardian,
        signature: GuardianSignature,
        message_hash: vector<u8>
    ): bool {
        let sig = guardian_signature::to_rsv(signature);
        as_bytes(self) == ecrecover(message_hash, sig)
    }

    /// Same as 'ecrecover' in EVM.
    fun ecrecover(message: vector<u8>, sig: vector<u8>): vector<u8> {
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

    #[test_only]
    public fun destroy(g: Guardian) {
        let Guardian { pubkey: _ } = g;
    }

    #[test_only]
    public fun to_bytes(value: Guardian): vector<u8> {
        let Guardian { pubkey } = value;
        bytes20::to_bytes(pubkey)
    }
}
