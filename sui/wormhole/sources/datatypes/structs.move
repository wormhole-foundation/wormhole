module wormhole::structs {
    use sui::tx_context::{Self, TxContext};

    friend wormhole::state;
    use wormhole::guardian_pubkey::{Self, GuardianPubkey};

    struct Signature has store, copy, drop {
        sig: vector<u8>,
        recovery_id: u8,
        guardian_index: u8,
    }

    struct Guardian has store, drop, copy {
        addr: GuardianPubkey
    }

    struct GuardianSet has store, copy, drop {
        index: u32,
        guardians: vector<Guardian>,
        expiration_time: u32,
    }

    public fun create_guardian(addr: vector<u8>): Guardian {
        Guardian {
            addr: guardian_pubkey::new(addr)
        }
    }

    public fun create_guardian_set(index: u32, guardians: vector<Guardian>): GuardianSet {
       GuardianSet {
            index: index,
            guardians: guardians,
            expiration_time: 0,
        }
    }

    public(friend) fun expire_guardian_set(guardian_set: &mut GuardianSet, delta: u32, ctx: &TxContext) {
        guardian_set.expiration_time = (tx_context::epoch(ctx) as u32) + delta;
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

    public fun get_address(guardian: &Guardian): GuardianPubkey {
        guardian.addr
    }

    public fun get_guardian_set_index(guardian_set: &GuardianSet): u32 {
        guardian_set.index
    }

    public fun get_guardians(guardian_set: &GuardianSet): vector<Guardian> {
        guardian_set.guardians
    }

    public fun get_guardian_set_expiry(guardian_set: &GuardianSet): u32 {
        guardian_set.expiration_time
    }
}
