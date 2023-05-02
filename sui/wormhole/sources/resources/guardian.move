// SPDX-License-Identifier: Apache 2

/// This module implements a `Guardian` that warehouses a 20-byte public key.
module wormhole::guardian {
    use std::vector::{Self};
    use sui::hash::{Self};
    use sui::ecdsa_k1::{Self};

    use wormhole::bytes20::{Self, Bytes20};
    use wormhole::guardian_signature::{Self, GuardianSignature};

    /// Guardian public key is all zeros.
    const E_ZERO_ADDRESS: u64 = 1;

    /// Container for 20-byte Guardian public key.
    struct Guardian has store {
        pubkey: Bytes20
    }

    /// Create new `Guardian` ensuring that the input is not all zeros.
    public fun new(pubkey: vector<u8>): Guardian {
        let data = bytes20::new(pubkey);
        assert!(bytes20::is_nonzero(&data), E_ZERO_ADDRESS);
        Guardian { pubkey: data }
    }

    /// Retrieve underlying 20-byte public key.
    public fun pubkey(self: &Guardian): Bytes20 {
        self.pubkey
    }

    /// Retrieve underlying 20-byte public key as `vector<u8>`.
    public fun as_bytes(self: &Guardian): vector<u8> {
        bytes20::data(&self.pubkey)
    }

    /// Verify that the recovered public key (using `ecrecover`) equals the one
    /// that exists for this Guardian with an elliptic curve signature and raw
    /// message that was signed.
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
            ecdsa_k1::decompress_pubkey(&ecdsa_k1::secp256k1_ecrecover(&sig, &message, 0));

        // `decompress_pubkey` returns 65 bytes. The last 64 bytes are what we
        // need to compute the Guardian's public key.
        vector::remove(&mut pubkey, 0);

        let hash = hash::keccak256(&pubkey);
        let guardian_pubkey = vector::empty<u8>();
        let (i, n) = (0, bytes20::length());
        while (i < n) {
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
