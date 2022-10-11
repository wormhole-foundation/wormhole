module wormhole::structs {
    use wormhole::myu32::{Self as u32, U32};
    use sui::tx_context::{Self, TxContext};

    friend wormhole::state;
    use wormhole::guardian_pubkey::{Self};

    struct Signature has store, copy, drop {
        sig: vector<u8>,
        recovery_id: u8,
        guardian_index: u8,
    }

    struct Guardian has store, drop, copy {
        address: guardian_pubkey::Address
    }

    struct GuardianSet has store, copy, drop {
        index:     U32,
        guardians: vector<Guardian>,
        expiration_time: U32,
    }

    public fun create_guardian(address: vector<u8>): Guardian {
        Guardian {
            address: guardian_pubkey::from_bytes(address)
        }
    }

    public fun create_guardian_set(index: U32, guardians: vector<Guardian>): GuardianSet {
       GuardianSet {
            index: index,
            guardians: guardians,
            expiration_time: u32::from_u64(0),
        }
    }

    public(friend) fun expire_guardian_set(guardian_set: &mut GuardianSet, delta: U32, ctx: &TxContext) {
        guardian_set.expiration_time = u32::from_u64(tx_context::epoch(ctx) + u32::to_u64(delta));
    }

    public fun unpack_signature(s: &Signature): (vector<u8>, u8, u8) {
        (s.sig, s.recovery_id, s.guardian_index)
    }

    public fun create_signature(
        sig: vector<u8>,
        recovery_id: u8,
        guardian_index: u8
    ): Signature {
        Signature{ sig, recovery_id, guardian_index }
    }

    public fun get_address(guardian: &Guardian): guardian_pubkey::Address {
        guardian.address
    }

    public fun get_guardian_set_index(guardian_set: &GuardianSet): U32 {
        guardian_set.index
    }

    public fun get_guardians(guardian_set: &GuardianSet): vector<Guardian> {
        guardian_set.guardians
    }

    public fun get_guardian_set_expiry(guardian_set: &GuardianSet): U32 {
        guardian_set.expiration_time
    }

}