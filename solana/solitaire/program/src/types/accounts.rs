//! Accounts.
//!
//! Solana provides a single primitive `AccountInfo` that represents an account on Solana. It
//! provides no information about what the account means however. This file provides a set of
//! types that describe different kinds of accounts to target.

use borsh::BorshSerialize;
use solana_program::{
    account_info::AccountInfo,
    program::invoke_signed,
    pubkey::Pubkey,
    system_instruction,
    sysvar::Sysvar as SolanaSysvar,
};
use std::ops::{
    Deref,
    DerefMut,
};

use crate::{
    processors::seeded::Owned,
    CreationLamports,
    Derive,
    ExecutionContext,
    Result,
};

/// A short alias for AccountInfo.
pub type Info<'r> = AccountInfo<'r>;

#[derive(Debug, Eq, PartialEq)]
pub enum AccountState {
    Initialized,
    Uninitialized,
    MaybeInitialized,
}

/// An account that is known to contain serialized data.
///
/// Note on const generics:
///
/// Solana's Rust version is JUST old enough that it cannot use constant variables in its default
/// parameter assignments. But these DO work in the consumption side so a user can still happily
/// use this type by writing for example:
///
/// Data<(), { AccountState::Uninitialized }>
#[rustfmt::skip]
pub struct Data<'r, T: Owned + Default, const IsInitialized: AccountState> (
    pub Box<Info<'r>>,
    pub T,
);

impl<'r, T: Owned + Default, const IsInitialized: AccountState> Deref
    for Data<'r, T, IsInitialized>
{
    type Target = T;
    fn deref(&self) -> &Self::Target {
        &self.1
    }
}

impl<'r, T: Owned + Default, const IsInitialized: AccountState> DerefMut
    for Data<'r, T, IsInitialized>
{
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.1
    }
}

impl<'r, T: Owned + Default, const IsInitialized: AccountState> Data<'r, T, IsInitialized> {
    /// Is the account already initialized / created
    pub fn is_initialized(&self) -> bool {
        **self.0.lamports.borrow() != 0
    }
}

pub struct Sysvar<'b, Var: SolanaSysvar>(pub AccountInfo<'b>, pub Var);

impl<'b, Var: SolanaSysvar> Deref for Sysvar<'b, Var> {
    type Target = Var;
    fn deref(&self) -> &Self::Target {
        &self.1
    }
}

impl<const Seed: &'static str> Derive<AccountInfo<'_>, Seed> {
    pub fn create(
        &self,
        ctx: &ExecutionContext,
        payer: &Pubkey,
        lamports: CreationLamports,
        space: usize,
        owner: &Pubkey,
    ) -> Result<()> {
        let ix = system_instruction::create_account(
            payer,
            self.0.key,
            lamports.amount(space),
            space as u64,
            owner,
        );
        let (_, bump_seed) = Pubkey::find_program_address(&[Seed.as_bytes()][..], ctx.program_id);
        invoke_signed(&ix, ctx.accounts, &[&[Seed.as_bytes(), &[bump_seed]]]).map_err(|e| e.into())
    }
}

impl<const Seed: &'static str, T: BorshSerialize + Owned + Default>
    Derive<Data<'_, T, { AccountState::Uninitialized }>, Seed>
{
    pub fn create(
        &self,
        ctx: &ExecutionContext,
        payer: &Pubkey,
        lamports: CreationLamports,
    ) -> Result<()> {
        // Get serialized struct size
        let size = self.0.try_to_vec().unwrap().len();
        let ix = system_instruction::create_account(
            payer,
            self.0 .0.key,
            lamports.amount(size),
            size as u64,
            ctx.program_id,
        );
        let (_, bump_seed) = Pubkey::find_program_address(&[Seed.as_bytes()][..], ctx.program_id);
        invoke_signed(&ix, ctx.accounts, &[&[Seed.as_bytes(), &[bump_seed]]]).map_err(|e| e.into())
    }
}
