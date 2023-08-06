//! This file contains several single-field wrapper structs. Each one represents a layer that must
//! be checked in order to parse a Solana account.
//!
//! These structs are always single field (or single + PhantomData) and so can be represented with
//! the transparent repr layout. When each layer is removed the data can be transmuted safely to
//! the layer below, allowing for optimized recursion.

use std::ops::{
    Deref,
    DerefMut,
};

#[repr(transparent)]
pub struct Mut<Next>(pub Next);

#[repr(transparent)]
pub struct MaybeMut<Next>(pub Next);

#[repr(transparent)]
pub struct Signer<Next>(pub Next);

#[repr(transparent)]
pub struct System<Next>(pub Next);

#[repr(transparent)]
pub struct Derive<Next, const SEED: &'static str>(pub Next);

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

impl<T> Deref for Mut<T> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl<T> DerefMut for Mut<T> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}

impl<T> Deref for MaybeMut<T> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl<T> DerefMut for MaybeMut<T> {
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

impl<T, const SEED: &'static str> Deref for Derive<T, SEED> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl<T, const SEED: &'static str> DerefMut for Derive<T, SEED> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}
