// SPDX-License-Identifier: Apache 2

/// This module implements a container that stores the token transfer amount
/// encoded in a Token Bridge message. These amounts are capped at 8 decimals.
/// This means that any amount of a coin whose metadata defines its decimals
/// as some value greater than 8, the encoded amount will be normalized to
/// eight decimals (which will lead to some residual amount after the transfer).
/// For inbound transfers, this amount will be denormalized (scaled by the same
/// decimal difference).
module token_bridge::normalized_amount {
    use sui::math::{Self};
    use wormhole::bytes::{Self};
    use wormhole::cursor::{Cursor};

    /// The amounts in the token bridge payload are truncated to 8 decimals
    /// in each of the contracts when sending tokens out, so there's no
    /// precision beyond 10^-8. We could preserve the original number of
    /// decimals when creating wrapped assets, and "untruncate" the amounts
    /// on the way out by scaling back appropriately. This is what most
    /// other chains do, but untruncating from 8 decimals to 18 decimals
    /// loses log2(10^10) ~ 33 bits of precision, which we cannot afford on
    /// Aptos (and Solana), as the coin type only has 64bits to begin with.
    /// Contrast with Ethereum, where amounts are 256 bits.
    /// So we cap the maximum decimals at 8 when creating a wrapped token.
    const MAX_DECIMALS: u8 = 8;

    /// Container holding the value decoded from a Token Bridge transfer.
    struct NormalizedAmount has store, copy, drop {
        value: u64
    }

    public fun cap_decimals(decimals: u8): u8 {
        if (decimals > MAX_DECIMALS) {
            MAX_DECIMALS
        } else {
            decimals
        }
    }

    public fun default(): NormalizedAmount {
        new(0)
    }

    public fun value(self: &NormalizedAmount): u64 {
        self.value
    }

    public fun take_value(norm: NormalizedAmount): u64 {
        let NormalizedAmount { value } = norm;
        value
    }

    public fun to_u256(norm: NormalizedAmount): u256 {
        (take_value(norm) as u256)
    }

    public fun from_raw(amount: u64, decimals: u8): NormalizedAmount {
        if (amount == 0) {
            default()
        } else if (decimals > MAX_DECIMALS) {
            new(amount / math::pow(10, decimals - MAX_DECIMALS))
        } else {
            new(amount)
        }
    }

    public fun to_raw(norm: NormalizedAmount, decimals: u8): u64 {
        let value = take_value(norm);

        if (value > 0 && decimals > MAX_DECIMALS) {
            value * math::pow(10, decimals - MAX_DECIMALS)
        } else {
            value
        }
    }

    public fun take_bytes(cur: &mut Cursor<u8>): NormalizedAmount {
        // Amounts are encoded with 32 bytes.
        let value = bytes::take_u256_be(cur);
        assert!(value < (1 << 64), 0);
        new((value as u64))
    }

    fun new(value: u64): NormalizedAmount {
        NormalizedAmount {
            value
        }
    }
}

#[test_only]
module token_bridge::normalized_amount_test {
    use token_bridge::normalized_amount;

    #[test]
    fun test_normalize_denormalize_amount() {
        let a = 12345678910111;
        let b = normalized_amount::from_raw(a, 9);
        let c = normalized_amount::to_raw(b, 9);
        assert!(c == 12345678910110, 0);

        let x = 12345678910111;
        let y = normalized_amount::from_raw(x, 5);
        let z = normalized_amount::to_raw(y, 5);
        assert!(z == x, 0);
    }
}
