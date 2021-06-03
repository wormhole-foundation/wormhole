#![feature(const_generics)]
#![feature(const_generics_defaults)]
#![allow(warnings)]

pub use rocksalt::*;

// Lacking:
//
// - Error is a lacking as its just a basic enum, maybe use errorcode.
// - Client generation incomplete.

// We need a few Solana things in scope in order to properly abstract Solana.
use solana_program::{
    account_info::{
        next_account_info,
        AccountInfo,
    },
    entrypoint,
    entrypoint::ProgramResult,
    instruction::AccountMeta,
    program::invoke_signed,
    program_error::ProgramError,
    program_pack::Pack,
    pubkey::Pubkey,
    rent::Rent,
    system_instruction,
    system_program,
    sysvar::{
        self,
        SysvarId,
    },
};

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
    slice::Iter,
    string::FromUtf8Error,
};

pub use borsh::{
    BorshDeserialize,
    BorshSerialize,
};

// Expose all submodules for consumption.
pub mod error;
pub mod macros;
pub mod processors;
pub mod types;

// We can also re-export a set of types at module scope, this defines the intended API we expect
// people to be able to use from top-level.
use crate::processors::seeded::Owned;
pub use crate::{
    error::{
        ErrBox,
        Result,
        SolitaireError,
    },
    macros::*,
    processors::{
        keyed::Keyed,
        peel::Peel,
        persist::Persist,
        seeded::Creatable,
    },
    types::*,
};

pub struct ExecutionContext<'a, 'b: 'a> {
    /// A reference to the program_id of the current program.
    pub program_id: &'a Pubkey,

    /// All accounts passed into the program
    pub accounts: &'a [AccountInfo<'b>],
}

/// Lamports to pay to an account being created
pub enum CreationLamports {
    Exempt,
    Amount(u64),
}

impl CreationLamports {
    /// Amount of lamports to be paid in account creation
    pub fn amount(self, size: usize) -> u64 {
        match self {
            CreationLamports::Exempt => Rent::default().minimum_balance(size),
            CreationLamports::Amount(v) => v,
        }
    }
}

pub trait Wrap {
    fn wrap(&self) -> Vec<AccountMeta>;
}

impl<T> Wrap for T
where
    T: ToAccounts,
{
    fn wrap(&self) -> Vec<AccountMeta> {
        self.to()
    }
}

impl<T> Wrap for Signer<T> {
    fn wrap(&self) -> Vec<AccountMeta> {
        todo!()
    }
}

impl<T, const Seed: &'static str> Wrap for Derive<T, Seed> {
    fn wrap(&self) -> Vec<AccountMeta> {
        todo!()
    }
}

impl<'a, T: BorshSerialize + Owned, const IsInitialized: bool, const Lazy: bool> Wrap
    for Data<'a, T, IsInitialized, Lazy>
{
    fn wrap(&self) -> Vec<AccountMeta> {
        todo!()
    }
}

pub trait InstructionContext<'a> {
    fn verify(&self, program_id: &Pubkey) -> Result<()> {
        Ok(())
    }

    fn deps(&self) -> Vec<Pubkey> {
        vec![]
    }
}

pub trait ToAccounts {
    fn to(&self) -> Vec<AccountMeta>;
}

/// Trait definition that describes types that can be constructed from a list of solana account
/// references. A list of dependent accounts is produced as a side effect of the parsing stage.
pub trait FromAccounts<'a, 'b: 'a, 'c> {
    fn from<T>(
        _: &'a Pubkey,
        _: &'c mut Iter<'a, AccountInfo<'b>>,
        _: &'a T,
    ) -> Result<(Self, Vec<Pubkey>)>
    where
        Self: Sized;
}
