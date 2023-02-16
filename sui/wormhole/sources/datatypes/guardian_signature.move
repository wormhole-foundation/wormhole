module wormhole::guardian_signature {
    use std::vector::{Self};

    const E_INVALID_RS_LENGTH: u64 = 0;

    struct GuardianSignature has store, copy, drop {
        rs: vector<u8>,
        recovery_id: u8,
        index: u8,
    }

    public fun new(
        rs: vector<u8>,
        recovery_id: u8,
        index: u8
    ): GuardianSignature {
        assert!(vector::length(&rs) == 64, E_INVALID_RS_LENGTH);
        GuardianSignature { rs, recovery_id, index }
    }

    public fun rs(self: &GuardianSignature): vector<u8> {
        self.rs
    }

    public fun recovery_id(self: &GuardianSignature): u8 {
        self.recovery_id
    }

    public fun index(self: &GuardianSignature): u8 {
        self.index
    }

    public fun index_as_u64(self: &GuardianSignature): u64 {
        (self.index as u64)
    }

    public fun destroy(signature: GuardianSignature): (vector<u8>, u8, u8) {
        let GuardianSignature { rs, recovery_id, index } = signature;
        (rs, recovery_id, index)
    }
}
