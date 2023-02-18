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

}
