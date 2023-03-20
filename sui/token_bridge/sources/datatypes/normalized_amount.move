/// Amounts in represented in token bridge VAAs are capped at 8 decimals. This
/// means that any amount that's given as having more decimals is truncated to 8
/// decimals. On the way out, these amount have to be scaled back to the
/// original decimal amount. This module defines `NormalizedAmount`, which
/// represents amounts that have been capped at 8 decimals.
///
/// The functions `normalize` and `denormalize` take care of convertion to/from
/// this type given the original amount's decimals.
module token_bridge::normalized_amount {
    use sui::math::{Self};
    use wormhole::cursor::{Cursor};
    use wormhole::bytes::{Self};

    struct NormalizedAmount has store, copy, drop {
        value: u64
    }

    public fun new(value: u64): NormalizedAmount {
        NormalizedAmount {
            value
        }
    }

    public fun default(): NormalizedAmount {
        new(0)
    }

    public fun value(self: &NormalizedAmount): u64 {
        self.value
    }

    public fun to_u256(self: &NormalizedAmount): u256 {
        (self.value as u256)
    }

    public fun from_u256(value: u256): NormalizedAmount {
        assert!(value < (1u256 << 64), 0);
        new((value as u64))
    }

    public fun from_raw(amount: u64, decimals: u8): NormalizedAmount {
        if (amount == 0) {
            default()
        } else {
            let normalized = {
                if (decimals > 8) {
                    amount / math::pow(10, decimals - 8)
                } else {
                    amount
                }
            };
            new(normalized)
        }
    }

    public fun to_raw(normalized: NormalizedAmount, decimals: u8): u64 {
        let NormalizedAmount { value } = normalized;
         if (value > 0 && decimals > 8) {
            value * math::pow(10, decimals - 8)
         } else {
            value
         }
    }

    public fun deserialize_be(cur: &mut Cursor<u8>): NormalizedAmount {
        // in the VAA wire format, amounts are 32 bytes.
        from_u256(bytes::take_u256_be(cur))
    }

    public fun serialize_be(buf: &mut vector<u8>, normalized: NormalizedAmount) {
        bytes::push_u256_be(buf, to_u256(&normalized))
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
