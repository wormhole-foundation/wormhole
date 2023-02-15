/// Guardian keys are EVM-style 20 byte addresses
/// That is, they are computed by taking the last 20 bytes of the keccak256
/// hash of their 64 byte secp256k1 public key.
module wormhole::guardian_pubkey {
    use sui::ecdsa_k1::{Self as ecdsa};
    use std::vector;
    use sui::ecdsa_k1::{keccak256};

    /// An error occurred while deserializing, for example due to wrong input size.
    const E_INVALID_FROM_EC_PUBKEY_LENGTH: u64 = 0;
    const E_INVALID_NEW_LENGTH: u64 = 1;

    struct GuardianPubkey has store, drop, copy {
        data: vector<u8>
    }

    /// Deserializes a raw byte sequence into an address.
    /// Aborts if the input is not 20 bytes long.
    public fun new(data: vector<u8>): GuardianPubkey {
        assert!(std::vector::length(&data) == 20, E_INVALID_NEW_LENGTH);
        GuardianPubkey { data }
    }

    /// Computes the address from a 64 byte public key.
    public fun from_ec_pubkey(pubkey: vector<u8>): GuardianPubkey {
        assert!(
            std::vector::length(&pubkey) == 64,
            E_INVALID_FROM_EC_PUBKEY_LENGTH
        );
        let hash = keccak256(&pubkey);
        let data = vector::empty<u8>();
        let i = 0;
        while (i < 20) {
            vector::push_back(&mut data, vector::pop_back(&mut hash));
            i = i + 1;
        };
        vector::reverse(&mut data);
        new(data)
    }

    /// Recovers the address from a signature and message.
    /// This is known as 'ecrecover' in EVM.
    public fun from_signature(
        message: vector<u8>,
        recovery_id: u8,
        sig: vector<u8>,
    ): GuardianPubkey {
        // sui's ecrecover function takes a 65 byte array (signature + recovery byte)
        vector::push_back(&mut sig, recovery_id);

        let pubkey =
            ecdsa::decompress_pubkey(&ecdsa::ecrecover(&sig, &message));

        // decompress_pubkey returns 65 bytes, the first byte is not relevant to
        // us, so we remove it
        vector::remove(&mut pubkey, 0);

        from_ec_pubkey(pubkey)
    }
}

#[test_only]
module wormhole::guardian_pubkey_test {
    use wormhole::guardian_pubkey;

    #[test]
    public fun from_pubkey_test() {
        // devnet guardian public key
        let pubkey = x"d4a4629979f0c9fa0f0bb54edf33f87c8c5a1f42c0350a30d68f7e967023e34e495a8ebf5101036d0fd66e3b0a8c7c61b65fceeaf487ab3cd1b5b7b50beb7970";
        let expected_address = guardian_pubkey::new(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe");

        let address = guardian_pubkey::from_ec_pubkey(pubkey);

        assert!(address == expected_address, 0);
    }

    #[test]
    public fun from_signature() {
        let sig = x"38535089d6eec412a00066f84084212316ee3451145a75591dbd4a1c2a2bff442223f81e58821bfa4e8ffb80a881daf7a37500b04dfa5719fff25ed4cec8dda3";
        let msg = x"43f3693ccdcb4400e1d1c5c8cec200153bd4b3d167e5b9fe5400508cf8717880";
        let addr = guardian_pubkey::from_signature(msg, 0x01, sig);
        let expected_addr = guardian_pubkey::new(x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe");
        assert!(addr == expected_addr, 0);
    }

}
