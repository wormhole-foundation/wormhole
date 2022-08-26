/// Guardian keys are EVM-style 20 byte addresses
/// That is, they are computed by taking the last 20 bytes of the keccak256
/// (sha3 256) hash of their 64 byte secp256k1 public key.
module Wormhole::guardian_pubkey {
    use 0x1::secp256k1::{
        ECDSARawPublicKey,
        ECDSASignature,
        ecdsa_raw_public_key_to_bytes,
        ecdsa_recover,
    };
    use 0x1::hash;
    use 0x1::vector;

    /// An error occurred while deserializing, for example due to wrong input size.
    const E_DESERIALIZE: u64 = 1;

    struct Address has key, store, drop, copy {
        bytes: vector<u8>
    }

    /// Deserializes a raw byte sequence into an address.
    /// Aborts if the input is not 20 bytes long.
    public fun from_bytes(bytes: vector<u8>): Address {
        assert!(std::vector::length(&bytes) == 20, std::error::invalid_argument(E_DESERIALIZE));
        Address { bytes }
    }

    /// Computes the address from a 64 byte public key.
    public fun from_pubkey(pubkey: &ECDSARawPublicKey): Address {
        let bytes = ecdsa_raw_public_key_to_bytes(pubkey);
        let hash = hash::sha3_256(bytes);
        let address = vector::empty<u8>();
        let i = 0;
        while (i < 20) {
            vector::push_back(&mut address, vector::pop_back(&mut hash));
            i = i + 1;
        };
        vector::reverse(&mut address);
        Address { bytes: address }
    }

    /// Recovers the address from a signature and message.
    /// This is known as 'ecrecover' in EVM.
    public fun from_signature(
        message: vector<u8>,
        recovery_id: u8,
        sig: &ECDSASignature,
    ): Address {
        let pubkey = ecdsa_recover(message, recovery_id, sig);
        let pubkey = std::option::extract(&mut pubkey);
        from_pubkey(&pubkey)
    }
}
