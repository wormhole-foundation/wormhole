//! Accounts.
//!
//! Solana provides a single primitive `AccountInfo` that represents an account on Solana. It
//! provides no information about what the account means however. This file provides a set of
//! types that describe different kinds of accounts to target.

use borsh::BorshSerialize;
use solana_program::{
    account_info::AccountInfo,
    program::{
        invoke,
        invoke_signed,
    },
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
    Keyed,
    Result,
};

/// A short alias for AccountInfo.
pub type Info<'r> = AccountInfo<'r>;

/// Represents an account's state in terms of initialization, which we use as
/// preconditions in our instructions.
///
/// * An [`Initialized`] account is one that has some data stored in it. If we
/// (the program) own the account, this means that we had written that data in
/// the first place. For security reasons, initialisation check should always be
/// in conjuction with an appropriate ownership check.
/// * An [`Uninitialized`] account is an account that has no data inside of it.
/// We may require an account to be [`Uninitialized`] when we want to initialize
/// it ourselves. Once initialized, an account's size and owner are immutable
/// (as guaranteed by the Solana runtime), so when we initialize a previously
/// uninitialized account, we will want to make sure to assign the right size
/// and owner.
/// * A [`MaybeInitialized`] account can be in either state. The instruction can
/// then query the state of the account and decide on a next step based on it.
#[derive(Debug, Eq, PartialEq)]
pub enum AccountState {
    Initialized,
    Uninitialized,
    MaybeInitialized,
}

/// Describes whether a cross-program invocation (CPI) should be
/// [`SignedWithSeeds`] or [`NotSigned`].
///
/// CPI calls inherit the signers of the original transaction, but the calling
/// program may optionally sign additional accounts using the program as the
/// signer. In this case, the signature is derived in a deterministic fashion by
/// using a set of seeds.
///
/// For more on program signed accounts, see the *[cpi docs].
///
/// [cpi docs]: https://docs.solana.com/developing/programming-model/calling-between-programs#program-signed-accounts
#[derive(PartialEq, PartialOrd, Eq, Ord, Debug, Hash)]
pub enum IsSigned<'a> {
    SignedWithSeeds(&'a [&'a [&'a [u8]]]),
    NotSigned,
}

use IsSigned::*;

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
pub struct Data<'r, T: Owned + Default, const IS_INITIALIZED: AccountState> (
    pub Box<Info<'r>>,
    pub T,
);

impl<'r, T: Owned + Default, const IS_INITIALIZED: AccountState> Deref
    for Data<'r, T, IS_INITIALIZED>
{
    type Target = T;
    fn deref(&self) -> &Self::Target {
        &self.1
    }
}

impl<'r, T: Owned + Default, const IS_INITIALIZED: AccountState> DerefMut
    for Data<'r, T, IS_INITIALIZED>
{
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.1
    }
}

impl<'r, T: Owned + Default> Data<'r, T, { AccountState::MaybeInitialized }> {
    /// Is the account already initialized / created
    pub fn is_initialized(&self) -> bool {
        !self.0.data.borrow().is_empty()
    }
}

pub struct Sysvar<'b, Var: SolanaSysvar>(pub AccountInfo<'b>, pub Var);

impl<'b, Var: SolanaSysvar> Deref for Sysvar<'b, Var> {
    type Target = Var;
    fn deref(&self) -> &Self::Target {
        &self.1
    }
}

impl<const SEED: &'static str> Derive<AccountInfo<'_>, SEED> {
    pub fn create(
        &self,
        ctx: &ExecutionContext,
        payer: &Pubkey,
        lamports: CreationLamports,
        space: usize,
        owner: &Pubkey,
    ) -> Result<()> {
        let (_, bump_seed) = Pubkey::find_program_address(&[SEED.as_bytes()][..], ctx.program_id);
        create_account(
            ctx,
            self.info(),
            payer,
            lamports,
            space,
            owner,
            SignedWithSeeds(&[&[SEED.as_bytes(), &[bump_seed]]]),
        )
    }
}

impl<const SEED: &'static str, T: BorshSerialize + Owned + Default>
    Derive<Data<'_, T, { AccountState::Uninitialized }>, SEED>
{
    pub fn create(
        &self,
        ctx: &ExecutionContext,
        payer: &Pubkey,
        lamports: CreationLamports,
    ) -> Result<()> {
        // Get serialized struct size
        let size = self.0.try_to_vec().unwrap().len();
        let (_, bump_seed) = Pubkey::find_program_address(&[SEED.as_bytes()][..], ctx.program_id);
        create_account(
            ctx,
            self.info(),
            payer,
            lamports,
            size,
            ctx.program_id,
            SignedWithSeeds(&[&[SEED.as_bytes(), &[bump_seed]]]),
        )
    }
}

/// Create an account.
///
/// This proceeds in the following order:
///
/// 1. Make sure the account has sufficient funds to cover the desired rent
/// period (top up if necessary).
/// 2. Allocate necessary size
/// 3. Assign ownership
///
/// We're not using the [`system_instruction::create_account`] instruction,
/// because it refuses to create an account if there's already money in the
/// account.
pub fn create_account(
    ctx: &ExecutionContext,
    account: &Info<'_>,
    payer: &Pubkey,
    lamports: CreationLamports,
    size: usize,
    owner: &Pubkey,
    seeds: IsSigned,
) -> Result<()> {
    let target_rent = lamports.amount(size)?;
    // top up account to target rent
    if account.lamports() < target_rent {
        let transfer_ix =
            system_instruction::transfer(payer, account.key, target_rent - account.lamports());
        invoke(&transfer_ix, ctx.accounts)?
    }
    // invoke is just a synonym for invoke_signed with an empty list
    let seeds = match seeds {
        SignedWithSeeds(v) => v,
        NotSigned => &[],
    };

    // allocate space
    let allocate_ix = system_instruction::allocate(account.key, size as u64);
    invoke_signed(&allocate_ix, ctx.accounts, seeds)?;

    // assign ownership
    let assign_ix = system_instruction::assign(account.key, owner);
    invoke_signed(&assign_ix, ctx.accounts, seeds)?;

    Ok(())
}
