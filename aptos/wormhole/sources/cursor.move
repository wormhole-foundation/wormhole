module wormhole::cursor {
    use std::vector;

    /// A cursor allows consuming a vector incrementally for parsing operations.
    /// It has no drop ability, and the only way to deallocate it is by calling the
    /// `destroy_empty` method, which will fail if the whole input hasn't been consumed.
    ///
    /// This setup statically guarantees that the parsing methods consume the
    /// full input.
    struct Cursor<T> {
        data: vector<T>,
    }

    /// Initialises a cursor from a vector.
    public fun init<T>(data: vector<T>): Cursor<T> {
        // reverse the array so we have access to the first element easily
        vector::reverse(&mut data);
        Cursor<T> {
            data,
        }
    }

    /// Destroys an empty cursor.
    /// Aborts if the cursor is not empty.
    public fun destroy_empty<T>(cur: Cursor<T>) {
        let Cursor { data } = cur;
        vector::destroy_empty(data);
    }

    /// Consumes the rest of the cursor (thus destroying it) and returns the
    /// remaining bytes.
    /// NOTE: Only use this function if you intend to consume the rest of the
    /// bytes. Since the result is a vector, which can be dropped, it is not
    /// possible to statically guarantee that the rest will be used.
    public fun rest<T>(cur: Cursor<T>): vector<T> {
        let Cursor { data } = cur;
        // re-reverse the data so it is in the same order as the original input
        vector::reverse(&mut data);
        data
    }

    /// Returns the first element of the cursor and advances it.
    public fun poke<T>(cur: &mut Cursor<T>): T {
        vector::pop_back(&mut cur.data)
    }
}
