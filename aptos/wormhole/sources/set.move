/// A set data structure.
module wormhole::set {
    use aptos_std::table::{Self, Table};

    /// Empty struct. Used as the value type in mappings to encode a set
    struct Unit has store, copy, drop {}

    /// A set containing elements of type `A` with support for membership
    /// checking.
    struct Set<phantom A: copy + drop> has store {
        elems: Table<A, Unit>
    }

    /// Create a new Set.
    public fun new<A: copy + drop>(): Set<A> {
        Set {
            elems: table::new()
        }
    }

    /// Add a new element to the set.
    /// Aborts if the element already exists
    public fun add<A: copy + drop>(set: &mut Set<A>, key: A) {
        table::add(&mut set.elems, key, Unit {})
    }

    /// Returns true iff `set` contains an entry for `key`.
    public fun contains<A: copy + drop>(set: &Set<A>, key: A): bool {
        table::contains(&set.elems, key)
    }

}
