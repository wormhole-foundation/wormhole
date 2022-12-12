/// Amounts in represented in token bridge VAAs are capped at 8 decimals. This
/// means that any amount that's given as having more decimals is truncated to 8
/// decimals. On the way out, these amount have to be scaled back to the
/// original decimal amount. This module defines `NormalizedAmount`, which
/// represents amounts that have been capped at 8 decimals.
///
/// The functions `normalize` and `denormalize` take care of convertion to/from
/// this type given the original amount's decimals.
module token_bridge::normalized_amount {
    use wormhole::cursor::Cursor;
    use wormhole::deserialize;
    use wormhole::serialize;

    struct NormalizedAmount has store, copy, drop {
        amount: u64
    }

    #[test_only]
    public fun get_amount(n: NormalizedAmount): u64 {
        n.amount
    }

    public fun normalize(amount: u64, decimals: u8): NormalizedAmount {
         if (decimals > 8) {
            let n = decimals - 8;
            while (n > 0) {
                amount = amount / 10;
                n = n - 1;
            }
         };
        NormalizedAmount { amount }
    }

    public fun denormalize(amount: NormalizedAmount, decimals: u8): u64 {
        let NormalizedAmount { amount } = amount;
         if (decimals > 8) {
            let n = decimals - 8;
            while (n > 0) {
                amount = amount * 10;
                n = n - 1;
            }
         };
         amount
    }

    public fun deserialize(cur: &mut Cursor<u8>): NormalizedAmount {
        // in the VAA wire format, amounts are 32 bytes.
        let amount = deserialize::deserialize_u256(cur);
        NormalizedAmount { amount: wormhole::u256::as_u64(amount) }
    }

    public fun serialize(buf: &mut vector<u8>, e: NormalizedAmount) {
        let NormalizedAmount { amount } = e;
        serialize::serialize_u256(buf, wormhole::u256::from_u64(amount))
    }
}

#[test_only]
module token_bridge::normalized_amount_test {
    use token_bridge::normalized_amount;

    #[test]
    fun test_normalize_denormalize_amount() {
        let a = 12345678910111;
        let b = normalized_amount::normalize(a, 9);
        let c = normalized_amount::denormalize(b, 9);
        assert!(c == 12345678910110, 0);

        let x = 12345678910111;
        let y = normalized_amount::normalize(x, 5);
        let z = normalized_amount::denormalize(y, 5);
        assert!(z == x, 0);
    }
}
