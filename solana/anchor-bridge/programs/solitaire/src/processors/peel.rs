//! Peeling.
//!
//! The accounts in Solitaire programs are defined via layers of types, when each layer is peeled
//! off it performs checks, parsing, and any other desired side-effect. The mechanism for this is
//! the peel trait, which defines a set of types that recursively construct the desired type.

use borsh::BorshDeserialize;
use solana_program::{
    pubkey::Pubkey,
    system_program,
    sysvar::{
        self,
        Sysvar as SolanaSysvar,
        SysvarId,
    },
};
use std::marker::PhantomData;

use crate::{
    processors::seeded::{
        AccountOwner,
        Owned,
    },
    types::*,
    Context,
    Result,
    SolitaireError,
};

/// Generic Peel trait. This provides a way to describe what each "peeled"
/// layer of our constraints should check.
pub trait Peel<'a, 'b: 'a, 'c> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self>
    where
        Self: Sized;
}

/// Peel a Derived Key
impl<'a, 'b: 'a, 'c, T: Peel<'a, 'b, 'c>, const Seed: &'static str> Peel<'a, 'b, 'c>
    for Derive<T, Seed>
{
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        // Attempt to Derive Seed
        let (derived, bump) = Pubkey::find_program_address(&[Seed.as_ref()], ctx.this);
        match derived == *ctx.info().key {
            true => T::peel(ctx).map(|v| Derive(v)),
            _ => Err(SolitaireError::InvalidDerive(*ctx.info().key).into()),
        }
    }
}

/// Peel a Signer.
impl<'a, 'b: 'a, 'c, T: Peel<'a, 'b, 'c>> Peel<'a, 'b, 'c> for Signer<T> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        match ctx.info().is_signer {
            true => T::peel(ctx).map(|v| Signer(v)),
            _ => Err(SolitaireError::InvalidSigner(*ctx.info().key).into()),
        }
    }
}

/// Expicitly depend upon the System account.
impl<'a, 'b: 'a, 'c, T: Peel<'a, 'b, 'c>> Peel<'a, 'b, 'c> for System<T> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        match true {
            true => T::peel(ctx).map(|v| System(v)),
            _ => panic!(),
        }
    }
}

/// Peel a Sysvar
impl<'a, 'b: 'a, 'c, T, Var> Peel<'a, 'b, 'c> for Sysvar<T, Var>
where
    T: Peel<'a, 'b, 'c>,
    Var: SolanaSysvar + SysvarId,
{
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        match <Var as SysvarId>::check_id(ctx.info().key) {
            true => T::peel(ctx).map(|v| Sysvar(v, PhantomData)),
            _ => Err(SolitaireError::InvalidSysvar(*ctx.info().key).into()),
        }
    }
}

/// This is our structural recursion base case, the trait system will stop generating new nested
/// calls here.
impl<'a, 'b: 'a, 'c> Peel<'a, 'b, 'c> for Info<'b> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        Ok(ctx.info().clone())
    }
}

/// This is our structural recursion base case, the trait system will stop generating new nested
/// calls here.
impl<'a, 'b: 'a, 'c, T: BorshDeserialize + Owned, const IsInitialized: bool, const Lazy: bool>
    Peel<'a, 'b, 'c> for Data<'b, T, IsInitialized, Lazy>
{
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        // If we're initializing the type, we should emit system/rent as deps.
        if !IsInitialized {
            ctx.deps.push(sysvar::rent::ID);
            ctx.deps.push(system_program::ID);
        }

        let data = T::try_from_slice(&mut *ctx.info().data.borrow_mut())?;

        if IsInitialized {
            match data.owner() {
                AccountOwner::This => {
                    if ctx.info().owner != ctx.this {
                        return Err(SolitaireError::InvalidOwner(*ctx.info().owner));
                    }
                }
                AccountOwner::Other(v) => {
                    if *ctx.info().owner != v {
                        return Err(SolitaireError::InvalidOwner(*ctx.info().owner));
                    }
                }
                AccountOwner::Any => {}
            };
        }

        Ok(Data(ctx.info().clone(), data))
    }
}
