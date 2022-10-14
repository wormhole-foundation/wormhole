/// Guardian keys are EVM-style 20 byte addresses
/// That is, they are computed by taking the last 20 bytes of the keccak256
/// hash of their 64 byte secp256k1 public key.
module wormhole::guardian_pubkey {
    use aptos_std::secp256k1::{
        ECDSARawPublicKey,
        ECDSASignature,
        ecdsa_raw_public_key_to_bytes,
        ecdsa_recover,
    };
    use std::vector;
    use wormhole::keccak256::keccak256;

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
        let hash = keccak256(bytes);
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

#[test_only]
module wormhole::guardian_pubkey_test {
    use wormhole::guardian_pubkey;
    use aptos_std::secp256k1::{
        ecdsa_raw_public_key_from_64_bytes,
        ecdsa_signature_from_bytes
    };

    #[test]
    public fun from_pubkey_test() {
        // devnet guardian public key
        let pubkey = x"d4a4629979f0c9fa0f0bb54edf33f87c8c5a1f42c0350a30d68f7e967023e34e495a8ebf5101036d0fd66e3b0a8c7c61b65fceeaf487ab3cd1b5b7b50beb7970";
        let pubkey = ecdsa_raw_public_key_from_64_bytes(pubkey);
        let expected_address = guardian_pubkey::from_bytes(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe");

        let address = guardian_pubkey::from_pubkey(&pubkey);

        assert!(address == expected_address, 0);
    }

    #[test]
    public fun from_signature() {
        let sig = ecdsa_signature_from_bytes(x"38535089d6eec412a00066f84084212316ee3451145a75591dbd4a1c2a2bff442223f81e58821bfa4e8ffb80a881daf7a37500b04dfa5719fff25ed4cec8dda3");
        let msg = x"43f3693ccdcb4400e1d1c5c8cec200153bd4b3d167e5b9fe5400508cf8717880";
        let addr = guardian_pubkey::from_signature(msg, 1, &sig);
        let expected_addr = guardian_pubkey::from_bytes(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe");
        assert!(addr == expected_addr, 0);
    }
}
