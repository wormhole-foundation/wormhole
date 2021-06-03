//! This file contains several single-field wrapper structs. Each one represents a layer that must
//! be checked in order to parse a Solana account.
//!
//! These structs are always single field (or single + PhantomData) and so can be represented with
//! the transparent repr layout. When each layer is removed the data can be transmuted safely to
//! the layer below, allowing for optimized recursion.

use std::{
    io::{
        ErrorKind,
        Write,
    },
    marker::PhantomData,
    ops::{
        Deref,
        DerefMut,
    },
};

use borsh::{
    BorshDeserialize,
    BorshSerialize,
};

#[repr(transparent)]
pub struct Signer<Next>(pub Next);

#[repr(transparent)]
pub struct System<Next>(pub Next);

#[repr(transparent)]
pub struct Sysvar<Next, Var>(pub Next, pub PhantomData<Var>);

#[repr(transparent)]
pub struct Derive<Next, const Seed: &'static str>(pub Next);

/// A tag for accounts that should be deserialized lazily.
#[allow(non_upper_case_globals)]
pub const Lazy: bool = true;

/// A tag for accounts that should be deserialized immediately (the default).
#[allow(non_upper_case_globals)]
pub const Strict: bool = false;

/// A tag for accounts that are expected to have already been initialized.
#[allow(non_upper_case_globals)]
pub const Initialized: bool = true;

/// A tag for accounts that must be uninitialized.
#[allow(non_upper_case_globals)]
pub const Uninitialized: bool = false;

// Several traits are required for types defined here, they cannot be defined in another file due
// to orphan instance limitations.

impl<T> Deref for Signer<T> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl<T> DerefMut for Signer<T> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}

impl<T, Var> Deref for Sysvar<T, Var> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl<T, Var> DerefMut for Sysvar<T, Var> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}

impl<T> Deref for System<T> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl<T> DerefMut for System<T> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}

impl<T, const Seed: &'static str> Deref for Derive<T, Seed> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl<T, const Seed: &'static str> DerefMut for Derive<T, Seed> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}
