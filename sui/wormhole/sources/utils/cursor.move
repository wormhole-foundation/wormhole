// SPDX-License-Identifier: Apache 2

/// This module implements a custom type that allows consuming a vector
/// incrementally for parsing operations. It has no drop ability, and the only
/// way to deallocate it is by calling the `destroy_empty` method, which will
/// fail if the whole input hasn't been consumed.
///
/// This setup statically guarantees that the parsing methods consume the full
/// input.
module wormhole::cursor {
    use std::vector::{Self};

    /// Container for the underlying `vector<u8>` data to be consumed.
    struct Cursor<T> {
        data: vector<T>,
    }

    /// Initialises a cursor from a vector.
    public fun new<T>(data: vector<T>): Cursor<T> {
        // reverse the array so we have access to the first element easily
        vector::reverse(&mut data);
        Cursor<T> { data }
    }

    /// Retrieve underlying data.
    public fun data<T>(self: &Cursor<T>): &vector<T> {
        &self.data
    }

    /// Check whether the underlying data is empty. This method is useful for
    /// iterating over a `Cursor` to exhaust its contents.
    public fun is_empty<T>(self: &Cursor<T>): bool {
        vector::is_empty(&self.data)
    }

    /// Destroys an empty cursor. This method aborts if the cursor is not empty.
    public fun destroy_empty<T>(cursor: Cursor<T>) {
        let Cursor { data } = cursor;
        vector::destroy_empty(data);
    }

    /// Consumes the rest of the cursor (thus destroying it) and returns the
    /// remaining bytes.
    ///
    /// NOTE: Only use this function if you intend to consume the rest of the
    /// bytes. Since the result is a vector, which can be dropped, it is not
    /// possible to statically guarantee that the rest will be used.
    public fun take_rest<T>(cursor: Cursor<T>): vector<T> {
        let Cursor { data } = cursor;
        // Because the data was reversed in initialization, we need to reverse
        // again so it is in the same order as the original input.
        vector::reverse(&mut data);
        data
    }

    /// Retrieve the first element of the cursor and advances it.
    public fun poke<T>(self: &mut Cursor<T>): T {
        vector::pop_back(&mut self.data)
    }
}
