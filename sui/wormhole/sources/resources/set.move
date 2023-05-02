// SPDX-License-Identifier: Apache 2

/// This module implements a custom type that resembles the set data structure.
/// `Set` leverages `sui::table` to store unique keys of the same type.
///
/// NOTE: Items added to this data structure cannot be removed.
module wormhole::set {
    use sui::table::{Self, Table};
    use sui::tx_context::{TxContext};

    /// Explicit error if key already exists in `Set`.
    const E_KEY_ALREADY_EXISTS: u64 = 0;
    /// Explicit error if key does not exist in `Set`.
    const E_KEY_NONEXISTENT: u64 = 1;

    /// Empty struct. Used as the value type in mappings to encode a set
    struct Empty has store, drop {}

    /// A set containing elements of type `T` with support for membership
    /// checking.
    struct Set<phantom T: copy + drop + store> has store {
        items: Table<T, Empty>
    }

    /// Create a new Set.
    public fun new<T: copy + drop + store>(ctx: &mut TxContext): Set<T> {
        Set { items: table::new(ctx) }
    }

    /// Add a new element to the set.
    /// Aborts if the element already exists
    public fun add<T: copy + drop + store>(self: &mut Set<T>, key: T) {
        assert!(!contains(self, key), E_KEY_ALREADY_EXISTS);
        table::add(&mut self.items, key, Empty {})
    }

    /// Returns true iff `set` contains an entry for `key`.
    public fun contains<T: copy + drop + store>(self: &Set<T>, key: T): bool {
        table::contains(&self.items, key)
    }

    public fun remove<T: copy + drop + store>(self: &mut Set<T>, key: T) {
        assert!(contains(self, key), E_KEY_NONEXISTENT);
        table::remove(&mut self.items, key);
    }

    #[test_only]
    public fun destroy<T: copy + drop + store>(set: Set<T>) {
        let Set { items } = set;
        table::drop(items);
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
