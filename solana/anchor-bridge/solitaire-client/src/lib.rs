#![feature(const_generics)]
#![feature(const_generics_defaults)]
#![allow(warnings)]

//! Client-specific code

pub use solana_program::pubkey::Pubkey;
pub use solana_sdk;

pub use solana_sdk::{
    instruction::{
        AccountMeta,
        Instruction,
    },
    signature::{
        Keypair,
        Signer as SolSigner,
    },
};

use borsh::BorshSerialize;

use solitaire::AccountState;
pub use solitaire::{
    Data,
    Derive,
    Keyed,
    Owned,
    Signer,
};

type StdResult<T, E> = std::result::Result<T, E>;

pub type ErrBox = Box<dyn std::error::Error>;

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

impl<'a, T, const IsInitialized: AccountState> Wrap for Data<'a, T, IsInitialized>
where
    T: BorshSerialize + Owned + Default,
{
    fn wrap(a: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox> {
        todo!();
    }
}

/// Trait used on client side to easily validate a program accounts + ix_data for a bare Solana call
pub trait ToInstruction {
    fn to_ix(
        self,
        program_id: Pubkey,
        ix_data: &[u8],
    ) -> StdResult<(Instruction, Vec<Keypair>), ErrBox>;
}
