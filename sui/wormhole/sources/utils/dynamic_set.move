/// This module recovers a Diem-style storage model where objects are collected
/// into a heterogeneous global storage, identified by their type.
///
/// Under the hood, it uses dynamic object fields, but set up in a way that the
/// key is derived from the value's type.
module wormhole::dynamic_set {
    use sui::dynamic_object_field as ofield;
    use sui::object::{UID};

    /// Wrap the value type. Avoids key collisions with other uses of dynamic
    /// fields.
    struct Wrapper<phantom Value> has copy, drop, store {
    }

    public fun add<Value: key + store>(
        object: &mut UID,
        value: Value,
    ) {
        ofield::add(object, Wrapper<Value>{}, value)
    }

    public fun borrow<Value: key + store>(
        object: &UID,
    ): &Value {
        ofield::borrow(object, Wrapper<Value>{})
    }

    public fun borrow_mut<Value: key + store>(
        object: &mut UID,
    ): &mut Value {
        ofield::borrow_mut(object, Wrapper<Value>{})
    }

    public fun remove<Value: key + store>(
        object: &mut UID,
    ): Value {
        ofield::remove(object, Wrapper<Value>{})
    }

    public fun exists_<Value: key + store>(
        object: &UID,
    ): bool {
        ofield::exists_<Wrapper<Value>>(object, Wrapper<Value>{})
    }
}
