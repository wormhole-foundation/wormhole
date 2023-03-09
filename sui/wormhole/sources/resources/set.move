/// A set data structure.
module wormhole::set {
    use sui::table::{Self, Table};
    use sui::tx_context::{TxContext};

    /// Empty struct. Used as the value type in mappings to encode a set
    struct Empty has store, copy, drop {}

    /// A set containing elements of type `T` with support for membership
    /// checking.
    struct Set<phantom T: copy + drop + store> has store {
        elems: Table<T, Empty>
    }

    /// Create a new Set.
    public fun new<T: copy + drop + store>(ctx: &mut TxContext): Set<T> {
        Set { elems: table::new(ctx) }
    }

    /// Add a new element to the set.
    /// Aborts if the element already exists
    public fun add<T: copy + drop + store>(set: &mut Set<T>, key: T) {
        table::add(&mut set.elems, key, Empty {})
    }

    /// Returns true iff `set` contains an entry for `key`.
    public fun contains<T: copy + drop + store>(set: &Set<T>, key: T): bool {
        table::contains(&set.elems, key)
    }

    #[test_only]
    public fun destroy<T: copy + drop + store>(s: Set<T>) {
        let Set { elems } = s;
        table::drop(elems);
    }

}

#[test_only]
module wormhole::set_tests {
    use sui::tx_context::{Self};

    use wormhole::set::{Self};

    #[test]
    public fun test_add_and_contains() {
        let ctx = &mut tx_context::dummy();

        let my_set = set::new(ctx);

        let (i, n) = (0, 256);
        while (i < n) {
            set::add(&mut my_set, i);
            i = i + 1;
        };

        // Check that the set has the values just added.
        let i = 0;
        while (i < n) {
            assert!(set::contains(&my_set, i), 0);
            i = i + 1;
        };

        // Check that these values that were not added are not in the set.
        while (i < 2 * n) {
            assert!(!set::contains(&my_set, i), 0);
            i = i + 1;
        };

        set::destroy(my_set);
    }
}
