//! Peeling.
//!
//! The accounts in Solitaire programs are defined via layers of types, when each layer is peeled
//! off it performs checks, parsing, and any other desired side-effect. The mechanism for this is
//! the peel trait, which defines a set of types that recursively construct the desired type.

use borsh::BorshDeserialize;
use solana_program::{
    instruction::AccountMeta,
    program_error::ProgramError,
    pubkey::Pubkey,
    system_program,
    sysvar::{
        self,
        Sysvar as SolanaSysvar,
        SysvarId,
    },
};
use std::{
    marker::PhantomData,
    slice::Iter,
};

use crate::{
    processors::seeded::{
        AccountOwner,
        Owned,
    },
    types::*,
    Context,
    FromAccounts,
    Result,
    SolitaireError,
};
use borsh::BorshSerialize;

/// Generic Peel trait. This provides a way to describe what each "peeled"
/// layer of our constraints should check.
pub trait Peel<'a, 'b: 'a, 'c> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self>
    where
        Self: Sized;

    fn deps() -> Vec<Pubkey>;

    fn persist(&self, program_id: &Pubkey) -> Result<()>;

    /// How many accounts from the on-chain account iterator
    /// constitute this peelable type? Important to customizefor
    /// multi-account types like FromAccounts implementors.
    fn partial_size_in_accounts() -> usize {
        1
    }

    /// Special method for turning an iterator into  AccountMetas for CPI use cases.
    fn to_partial_cpi_metas(_: &'c mut Iter<Info<'b>>) -> Result<Vec<AccountMeta>>;
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
            _ => Err(SolitaireError::InvalidDerive(*ctx.info().key, derived).into()),
        }
    }

    fn deps() -> Vec<Pubkey> {
        T::deps()
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        T::persist(self, program_id)
    }

    fn to_partial_cpi_metas(infos: &'c mut Iter<Info<'b>>) -> Result<Vec<AccountMeta>> {
        T::to_partial_cpi_metas(infos)
    }
}

/// Peel a Mutable key.
impl<'a, 'b: 'a, 'c, T: Peel<'a, 'b, 'c>> Peel<'a, 'b, 'c> for Mut<T> {
    fn peel<I>(mut ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        ctx.immutable = false;
        match ctx.info().is_writable {
            true => T::peel(ctx).map(|v| Mut(v)),
            _ => Err(SolitaireError::InvalidMutability(*ctx.info().key).into()),
        }
    }

    fn deps() -> Vec<Pubkey> {
        T::deps()
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        T::persist(self, program_id)
    }
    fn to_partial_cpi_metas(infos: &'c mut Iter<Info<'b>>) -> Result<Vec<AccountMeta>> {
        T::to_partial_cpi_metas(infos)
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

    fn deps() -> Vec<Pubkey> {
        T::deps()
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        T::persist(self, program_id)
    }

    fn to_partial_cpi_metas(infos: &'c mut Iter<Info<'b>>) -> Result<Vec<AccountMeta>> {
        T::to_partial_cpi_metas(infos)
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

    fn deps() -> Vec<Pubkey> {
        T::deps()
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        T::persist(self, program_id)
    }

    fn to_partial_cpi_metas(infos: &'c mut Iter<Info<'b>>) -> Result<Vec<AccountMeta>> {
        T::to_partial_cpi_metas(infos)
    }
}

/// Peel a Sysvar
impl<'a, 'b: 'a, 'c, Var> Peel<'a, 'b, 'c> for Sysvar<'b, Var>
where
    Var: SolanaSysvar,
{
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        match Var::check_id(ctx.info().key) {
            true => Ok(Sysvar(
                ctx.info().clone(),
                Var::from_account_info(ctx.info())?,
            )),
            _ => Err(SolitaireError::InvalidSysvar(*ctx.info().key).into()),
        }
    }

    fn deps() -> Vec<Pubkey> {
        vec![]
    }

    fn persist(&self, _program_id: &Pubkey) -> Result<()> {
        Ok(())
    }

    fn to_partial_cpi_metas(infos: &'c mut Iter<Info<'b>>) -> Result<Vec<AccountMeta>> {
        Info::to_partial_cpi_metas(infos)
    }
}

/// This is our structural recursion base case, the trait system will stop generating new nested
/// calls here.
impl<'a, 'b: 'a, 'c> Peel<'a, 'b, 'c> for Info<'b> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        if ctx.immutable && ctx.info().is_writable {
            return Err(SolitaireError::InvalidMutability(*ctx.info().key).into());
        }

        Ok(ctx.info().clone())
    }
    fn deps() -> Vec<Pubkey> {
        vec![]
    }
    fn persist(&self, _program_id: &Pubkey) -> Result<()> {
        Ok(())
    }

    fn to_partial_cpi_metas(infos: &'c mut Iter<Info<'b>>) -> Result<Vec<AccountMeta>> {
        let acc = infos
            .next()
            .ok_or_else(|| SolitaireError::ProgramError(ProgramError::AccountDataTooSmall))?;
        let meta = if acc.is_writable {
            AccountMeta::new(acc.key.clone(), acc.is_signer)
        } else {
            AccountMeta::new_readonly(acc.key.clone(), acc.is_signer)
        };

        Ok(vec![meta])
    }
}

/// This is our structural recursion base case, the trait system will stop generating new nested
/// calls here.
impl<
        'a,
        'b: 'a,
        'c,
        T: BorshDeserialize + BorshSerialize + Owned + Default,
        const IsInitialized: AccountState,
    > Peel<'a, 'b, 'c> for Data<'b, T, IsInitialized>
{
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        if ctx.immutable && ctx.info().is_writable {
            return Err(SolitaireError::InvalidMutability(*ctx.info().key).into());
        }

        // If we're initializing the type, we should emit system/rent as deps.
        let (initialized, data): (bool, T) = match IsInitialized {
            AccountState::Uninitialized => {
                if **ctx.info().lamports.borrow() != 0 {
                    return Err(SolitaireError::AlreadyInitialized(*ctx.info().key));
                }
                (false, T::default())
            }
            AccountState::Initialized => {
                (true, T::try_from_slice(&mut *ctx.info().data.borrow_mut())?)
            }
            AccountState::MaybeInitialized => {
                if **ctx.info().lamports.borrow() == 0 {
                    (false, T::default())
                } else {
                    (true, T::try_from_slice(&mut *ctx.info().data.borrow_mut())?)
                }
            }
        };

        if initialized {
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

    fn deps() -> Vec<Pubkey> {
        if IsInitialized == AccountState::Initialized {
            return vec![];
        }

        vec![sysvar::rent::ID, system_program::ID]
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        // TODO: Introduce Mut<> to solve the check we really want to make here.
        if self.0.owner != program_id {
            return Ok(());
        }

        // It is also a malformed program to attempt to write to a non-writeable account.
        if !self.0.is_writable {
            return Ok(());
        }

        self.1.serialize(&mut *self.0.data.borrow_mut())?;

        Ok(())
    }

    fn to_partial_cpi_metas(infos: &'c mut Iter<Info<'b>>) -> Result<Vec<AccountMeta>> {
        Info::to_partial_cpi_metas(infos)
    }
}

impl<'a, 'b: 'a, 'c, T: FromAccounts<'a, 'b, 'c>> Peel<'a, 'b, 'c>
    for CPICall<T>
{
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self>
    where
        Self: Sized,
    {
        let xprog = ctx
            .iter
            .next()
            .ok_or_else(|| SolitaireError::ProgramError(ProgramError::AccountDataTooSmall))?;

        let xprog_accounts = T::to_cpi_metas(ctx.iter)?;

        if xprog_accounts.len() != T::size_in_accounts() {
            return Err(SolitaireError::ProgramError(
                ProgramError::AccountDataTooSmall,
            ));
        }
        Ok(Self {
            xprog_id: xprog.key.clone(),
            xprog_accounts,
            callee_type: PhantomData,
        })
    }

    fn deps() -> Vec<Pubkey> {
        vec![system_program::id()]
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        // Persisting cross accounts is not our business
        Ok(())
    }

    fn to_partial_cpi_metas(infos: &'c mut Iter<Info<'b>>) -> Result<Vec<AccountMeta>> {
        T::to_cpi_metas(infos)
    }

    fn partial_size_in_accounts() -> usize {
        T::size_in_accounts() + 1 // Nested type size + 1 for cross program ID
    }
}

impl<'a, 'b: 'a, 'c, T: FromAccounts<'a, 'b, 'c>> Peel<'a, 'b, 'c> for Many<T> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self>
    where
        Self: Sized,
    {
        Ok(Self(T::from(ctx.this, ctx.iter, ctx.data)?))
    }

    fn deps() -> Vec<Pubkey> {
        // deps handled in peel() using <T as FromAccounts>::from()
        vec![]
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        self.persist(program_id)
    }

    fn to_partial_cpi_metas(infos: &'c mut Iter<Info<'b>>) -> Result<Vec<AccountMeta>> {
        T::to_cpi_metas(infos)
    }

    fn partial_size_in_accounts() -> usize {
        T::size_in_accounts()
    }
}
