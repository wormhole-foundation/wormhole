// SPDX-License-Identifier: Apache 2

/// This module implements a custom type representing a Guardian's signature
/// with recovery ID of a particular hashed VAA message body. The components of
/// `GuardianSignature` are used to perform public key recovery using ECDSA.
module wormhole::guardian_signature {
    use std::vector::{Self};

    use wormhole::bytes32::{Self, Bytes32};

    /// Container for elliptic curve signature parameters and Guardian index.
    struct GuardianSignature has store, drop {
        r: Bytes32,
        s: Bytes32,
        recovery_id: u8,
        index: u8,
    }

    /// Create new `GuardianSignature`.
    public fun new(
        r: Bytes32,
        s: Bytes32,
        recovery_id: u8,
        index: u8
    ): GuardianSignature {
        GuardianSignature { r, s, recovery_id, index }
    }

    /// 32-byte signature parameter R.
    public fun r(self: &GuardianSignature): Bytes32 {
        self.r
    }

    /// 32-byte signature parameter S.
    public fun s(self: &GuardianSignature): Bytes32 {
        self.s
    }

    /// Signature recovery ID.
    public fun recovery_id(self: &GuardianSignature): u8 {
        self.recovery_id
    }

    /// Guardian index.
    public fun index(self: &GuardianSignature): u8 {
        self.index
    }

    /// Guardian index as u64.
    public fun index_as_u64(self: &GuardianSignature): u64 {
        (self.index as u64)
    }

    /// Serialize elliptic curve paramters as `vector<u8>` of length == 65 to be
    /// consumed by `ecdsa_k1` for public key recovery.
    public fun to_rsv(gs: GuardianSignature): vector<u8> {
        let GuardianSignature { r, s, recovery_id, index: _ } = gs;
        let out = vector::empty();
        vector::append(&mut out, bytes32::to_bytes(r));
        vector::append(&mut out, bytes32::to_bytes(s));
        vector::push_back(&mut out, recovery_id);
        out
    }
}
