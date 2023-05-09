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
    use wormhole::bytes32::{Self};
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

    public fun max_decimals(): u8 {
        MAX_DECIMALS
    }

    /// Utility function to cap decimal amount to 8.
    public fun cap_decimals(decimals: u8): u8 {
        if (decimals > MAX_DECIMALS) {
            MAX_DECIMALS
        } else {
            decimals
        }
    }

    /// Create new `NormalizedAmount` of zero.
    public fun default(): NormalizedAmount {
        new(0)
    }

    /// Retrieve underlying value.
    public fun value(self: &NormalizedAmount): u64 {
        self.value
    }

    /// Retrieve underlying value as `u256`.
    public fun to_u256(norm: NormalizedAmount): u256 {
        (take_value(norm) as u256)
    }

    /// Create new `NormalizedAmount` using raw amount and specified decimals.
    public fun from_raw(amount: u64, decimals: u8): NormalizedAmount {
        if (amount == 0) {
            default()
        } else if (decimals > MAX_DECIMALS) {
            new(amount / math::pow(10, decimals - MAX_DECIMALS))
        } else {
            new(amount)
        }
    }

    /// Denormalize `NormalizedAmount` using specified decimals.
    public fun to_raw(norm: NormalizedAmount, decimals: u8): u64 {
        let value = take_value(norm);

        if (value > 0 && decimals > MAX_DECIMALS) {
            value * math::pow(10, decimals - MAX_DECIMALS)
        } else {
            value
        }
    }

    /// Transform `NormalizedAmount` to serialized (big-endian) u256.
    public fun to_bytes(norm: NormalizedAmount): vector<u8> {
        bytes32::to_bytes(bytes32::from_u256_be(to_u256(norm)))
    }

    /// Read 32 bytes from `Cursor` and deserialize to u64, ensuring no
    /// overflow.
    public fun take_bytes(cur: &mut Cursor<u8>): NormalizedAmount {
        // Amounts are encoded with 32 bytes.
        new(bytes32::to_u64_be(bytes32::take_bytes(cur)))
    }

    fun new(value: u64): NormalizedAmount {
        NormalizedAmount {
            value
        }
    }

    fun take_value(norm: NormalizedAmount): u64 {
        let NormalizedAmount { value } = norm;
        value
    }
}

#[test_only]
module token_bridge::normalized_amount_test {
    use wormhole::bytes::{Self};
    use wormhole::cursor::{Self};

    use token_bridge::normalized_amount::{Self};

    #[test]
    fun test_from_and_to_raw() {
        // Use decimals > 8 to check truncation.
        let decimals = 9;
        let raw_amount = 12345678910111;
        let normalized = normalized_amount::from_raw(raw_amount, decimals);
        let denormalized = normalized_amount::to_raw(normalized, decimals);
        assert!(denormalized == 10 * (raw_amount / 10), 0);

        // Use decimals <= 8 to check raw amount recovery.
        let decimals = 5;
        let normalized = normalized_amount::from_raw(raw_amount, decimals);
        let denormalized = normalized_amount::to_raw(normalized, decimals);
        assert!(denormalized == raw_amount, 0);
    }

    #[test]
    fun test_take_bytes() {
        let cur =
            cursor::new(
                x"000000000000000000000000000000000000000000000000ffffffffffffffff"
            );

        let norm = normalized_amount::take_bytes(&mut cur);
        assert!(
            normalized_amount::value(&norm) == ((1u256 << 64) - 1 as u64),
            0
        );

        // Clean up.
        cursor::destroy_empty(cur);
    }

    #[test]
    #[expected_failure(abort_code = wormhole::bytes32::E_U64_OVERFLOW)]
    fun test_cannot_take_bytes_overflow() {
        let encoded_overflow =
            x"0000000000000000000000000000000000000000000000010000000000000000";

        let amount = {
            let cur = cursor::new(encoded_overflow);
            let value = bytes::take_u256_be(&mut cur);
            cursor::destroy_empty(cur);
            value
        };
        assert!(amount == (1 << 64), 0);

        let cur = cursor::new(encoded_overflow);

        // You shall not pass!
        normalized_amount::take_bytes(&mut cur);

        abort 42
    }
}
