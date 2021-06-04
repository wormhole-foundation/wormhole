#![feature(const_generics)]
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
    instruction::{
        AccountMeta,
        Instruction,
    },
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
use solana_sdk::signature::{
    Keypair,
    Signer as SolSigner,
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

type StdResult<T, E> = std::result::Result<T, E>;

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

/// The sum type for clearly specifying the accounts required on client side.
pub enum AccEntry {
    /// Least privileged account.
    Unprivileged(Pubkey),

    /// Accounts that need to sign a Solana call
    Signer(Keypair),
    SignerRO(Keypair),

    /// Program addresses for privileged/unprivileged cross calls
    CPIProgram(Pubkey),
    CPIProgramSigner(Keypair),

    /// Key decided by Wrap implementation
    Sysvar,
    Derived(Pubkey),
    DerivedRO(Pubkey),
}

/// Types implementing Wrap are those that can be turned into a
/// partial account vector tha
/// payload.
pub trait Wrap {
    fn wrap(_: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox>;

    /// If the implementor wants to sign using other AccEntry
    /// variants, they should override this.
    fn keypair(a: AccEntry) -> Option<Keypair> {
        use AccEntry::*;
        match a {
            Signer(pair) => Some(pair),
            SignerRO(pair) => Some(pair),
            _other => None,
        }
    }
}

impl<'a, 'b: 'a, T> Wrap for Signer<T>
where
    T: Keyed<'a, 'b>,
{
    fn wrap(a: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox> {
        use AccEntry::*;
        match a {
            Signer(pair) => Ok(vec![AccountMeta::new(pair.pubkey(), true)]),
            SignerRO(pair) => Ok(vec![AccountMeta::new_readonly(pair.pubkey(), true)]),
            other => Err(format!(
                "{} must be passed as Signer or SignerRO",
                std::any::type_name::<Self>()
            )
            .into()),
        }
    }
}

impl<'a, 'b: 'a, T, const Seed: &'static str> Wrap for Derive<T, Seed> {
    fn wrap(a: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox> {
        match a {
            AccEntry::Derived(program_id) => {
                let (k, extra_seed) = Pubkey::find_program_address(&[Seed.as_bytes()], &program_id);

                Ok(vec![AccountMeta::new(k, false)])
            }
            AccEntry::DerivedRO(program_id) => {
                let (k, extra_seed) = Pubkey::find_program_address(&[Seed.as_bytes()], &program_id);

                Ok(vec![AccountMeta::new_readonly(k, false)])
            }
            other => Err(format!(
                "{} must be passed as Derived or DerivedRO",
                std::any::type_name::<Self>()
            )
            .into()),
        }
    }
}

impl<'a, T: BorshSerialize + Owned + Default, const IsInitialized: AccountState> Wrap
    for Data<'a, T, IsInitialized>
{
    fn wrap(a: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox> {
        todo!();
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

/// Trait used on client side to easily validate a program accounts + ix_data for a bare Solana call
pub trait ToInstruction {
    fn to_ix(self, program_id: Pubkey, ix_data: &[u8]) -> StdResult<(Instruction, Vec<Keypair>), ErrBox>;
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
