module wormhole::guardian_signature {
    use std::vector::{Self};

    use wormhole::bytes32::{Self, Bytes32};

    const E_INVALID_BUFFER_SIZE: u64 = 0;

    struct GuardianSignature has store, copy, drop {
        r: Bytes32,
        s: Bytes32,
        recovery_id: u8,
        index: u8,
    }

    public fun new(
        r: Bytes32,
        s: Bytes32,
        recovery_id: u8,
        index: u8
    ): GuardianSignature {
        GuardianSignature { r, s, recovery_id, index }
    }

    public fun r(self: &GuardianSignature): Bytes32 {
        self.r
    }

    public fun s(self: &GuardianSignature): Bytes32 {
        self.s
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

    public fun to_rsv(gs: GuardianSignature): vector<u8> {
        let GuardianSignature { r, s, recovery_id, index: _ } = gs;
        let out = vector::empty();
        vector::append(&mut out, bytes32::to_bytes(r));
        vector::append(&mut out, bytes32::to_bytes(s));
        vector::push_back(&mut out, recovery_id);
        out
    }
}
