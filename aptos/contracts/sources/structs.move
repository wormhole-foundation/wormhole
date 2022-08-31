module wormhole::structs {
    use wormhole::u32::{Self, U32};
    use 0x1::secp256k1::{Self};

    friend wormhole::state;
    friend wormhole::vaa;
    friend wormhole::wormhole;
    use wormhole::guardian_pubkey::{Self};

    struct Signature has key, store, copy, drop {
        sig: secp256k1::ECDSASignature,
        recovery_id: u8,
        guardian_index: u8,
    }

    struct Guardian has key, store, drop, copy {
        address: guardian_pubkey::Address
    }

    struct GuardianSet has key, store, copy, drop {
        index:     U32,
        guardians: vector<Guardian>,
        expiration_time: U32,
    }

    public fun create_guardian(address: vector<u8>): Guardian {
        Guardian{
            address: guardian_pubkey::from_bytes(address)
        }
    }

    public fun create_guardian_set(index: U32, guardians: vector<Guardian>): GuardianSet {
        GuardianSet{
            index: index,
            guardians: guardians,
            expiration_time: u32::from_u64(0),
        }
    }

    public(friend) fun expire_guardian_set(_guardian_set: &mut GuardianSet) {
        // TODO - right now U32 addition not supported
        //guardian_set.expiration_time = timestamp::now_seconds() + 86400;
    }

    public fun unpack_signature(s: &Signature): (secp256k1::ECDSASignature, u8, u8) {
        (s.sig, s.recovery_id, s.guardian_index)
    }

    public fun create_signature(
        sig: secp256k1::ECDSASignature,
        recovery_id: u8,
        guardian_index: u8
    ): Signature {
        Signature{ sig, recovery_id, guardian_index }
    }

    public fun get_address(guardian: Guardian): guardian_pubkey::Address {
        guardian.address
    }

    public fun get_guardian_set_index(guardian_set: GuardianSet): U32 {
        guardian_set.index
    }

    public fun get_guardians(guardian_set: GuardianSet): vector<Guardian> {
        guardian_set.guardians
    }

    public fun get_guardian_set_expiry(guardian_set: GuardianSet): U32 {
        guardian_set.expiration_time
    }

}
