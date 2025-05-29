use super::keyed::Keyed;
use crate::{
    create_account,
    AccountState,
    CreationLamports,
    Data,
    Derive,
    ExecutionContext,
    IsSigned::*,
    Result,
    SolitaireError,
};
use borsh::BorshSerialize;
use solana_program::{
    entrypoint::ProgramResult,
    instruction::Instruction,
    program::invoke_signed,
    pubkey::Pubkey,
};

pub trait AccountSize {
    fn size(&self) -> usize;
}

pub enum AccountOwner {
    This,
    Other(Pubkey),
    OneOf(Vec<Pubkey>),
    Any,
}

/// The ownership story:
/// We (solitaire) require every account to have an owner specified for two reasons:
/// 1. Security checks (i.e. solitaire checks that the account is owned by whoever we expect)
/// 2. Account creation
///
/// Solitaire programs create many accounts, most of which are owned by the program itself.
/// However, some accounts are owned by external programs. When performing security
/// checks, the ownership is expressed as a constraint on the account, and this
/// constraint may allow multiple potential owners.
/// But when creating an account, we need to know exactly who the owner is.
///
/// An example of this is a token account, which, when checking its owner, we
/// check that it's owned by _either_ the SPL token program, or the
/// Token2022 program. However, when creating a token account, we need to
/// know exactly which program is the expected owner.
///
/// To this end, we have two traits:
/// - `SingleOwned` is for accounts that have a single owner
/// - `MultiOwned` is for accounts that can have multiple potential owners
///
/// This stratification allows us then have two separate creation flows:
/// `Creatable` for accounts with a single owner, and `CreatableWithOwner` for
/// accounts whose owner is one of a set of owners. This way, assuming that only the
/// appropriate trait is implemented for a type, the compiler statically
/// guarantees that the appropriate creation flow is used (.create vs .create_with_owner).
pub trait Owned {
    fn owner(&self) -> AccountOwner;
}

/// A trait for accounts that may have a single owner. Most accounts are like this.
pub trait SingleOwned: Owned {
    fn owner_pubkey(&self, program_id: &Pubkey) -> Result<Pubkey> {
        match self.owner() {
            AccountOwner::This => Ok(*program_id),
            AccountOwner::Other(v) => Ok(v),
            AccountOwner::OneOf(_) => Err(SolitaireError::AmbiguousOwner),
            AccountOwner::Any => Err(SolitaireError::AmbiguousOwner),
        }
    }
}

/// A trait for accounts that may have multiple potential owners. This is the
/// equivalent of an `InterfaceAccount` in anchor.
pub trait MultiOwned: Owned {
    fn owners_pubkeys(&self, program_id: &Pubkey) -> Result<Vec<Pubkey>> {
        match self.owner() {
            AccountOwner::This => Ok(vec![*program_id]),
            AccountOwner::Other(v) => Ok(vec![v]),
            AccountOwner::OneOf(v) => Ok(v),
            AccountOwner::Any => Err(SolitaireError::AmbiguousOwner),
        }
    }
}

impl<'a, T: Owned + Default, const IS_INITIALIZED: AccountState> Owned
    for Data<'a, T, IS_INITIALIZED>
{
    fn owner(&self) -> AccountOwner {
        self.1.owner()
    }
}

impl<'a, T: SingleOwned + Default, const IS_INITIALIZED: AccountState> SingleOwned
    for Data<'a, T, IS_INITIALIZED>
{
}

impl<'a, T: MultiOwned + Default, const IS_INITIALIZED: AccountState> MultiOwned
    for Data<'a, T, IS_INITIALIZED>
{
}

pub trait Seeded<I> {
    fn seeds(accs: I) -> Vec<Vec<u8>>;

    fn self_seeds(&self, accs: I) -> Vec<Vec<u8>> {
        Self::seeds(accs)
    }

    fn key(accs: I, program_id: &Pubkey) -> Pubkey {
        let seeds = Self::seeds(accs);
        let s: Vec<&[u8]> = seeds.iter().map(|item| item.as_slice()).collect();
        let seed_slice = s.as_slice();
        let (addr, _) = Pubkey::find_program_address(seed_slice, program_id);

        addr
    }

    fn bumped_seeds(accs: I, program_id: &Pubkey) -> Vec<Vec<u8>> {
        let mut seeds = Self::seeds(accs);
        let s: Vec<&[u8]> = seeds.iter().map(|item| item.as_slice()).collect();
        let seed_slice = s.as_slice();
        let (_, bump_seed) = Pubkey::find_program_address(seed_slice, program_id);
        seeds.push(vec![bump_seed]);

        seeds
    }

    fn self_bumped_seeds(&self, accs: I, program_id: &Pubkey) -> Vec<Vec<u8>> {
        Self::bumped_seeds(accs, program_id)
    }

    fn verify_derivation<'a, 'b: 'a>(&'a self, program_id: &'a Pubkey, accs: I) -> Result<()>
    where
        Self: Keyed<'a, 'b>,
    {
        let seeds = Self::seeds(accs);
        let s: Vec<&[u8]> = seeds.iter().map(|item| item.as_slice()).collect();
        let seed_slice = s.as_slice();

        let (derived, _bump) = Pubkey::find_program_address(seed_slice, program_id);
        if &derived == self.info().key {
            Ok(())
        } else {
            Err(SolitaireError::InvalidDerive(*self.info().key, derived))
        }
    }
}

pub trait Creatable<'a, I> {
    fn create(
        &'a self,
        accs: I,
        ctx: &'a ExecutionContext,
        payer: &'a Pubkey,
        lamports: CreationLamports,
    ) -> Result<()>;
}

pub trait CreatableWithOwner<'a, I> {
    fn create_with_owner(
        &'a self,
        owner: &Pubkey,
        accs: I,
        ctx: &'a ExecutionContext,
        payer: &'a Pubkey,
        lamports: CreationLamports,
    ) -> Result<()>;
}

impl<T: BorshSerialize + Owned + Default, const IS_INITIALIZED: AccountState> AccountSize
    for Data<'_, T, IS_INITIALIZED>
{
    fn size(&self) -> usize {
        self.1.try_to_vec().unwrap().len()
    }
}

impl<'a, 'b: 'a, K, T: AccountSize + Seeded<K> + Keyed<'a, 'b> + SingleOwned> Creatable<'a, K>
    for T
{
    fn create(
        &'a self,
        accs: K,
        ctx: &'a ExecutionContext<'_, '_>,
        payer: &'a Pubkey,
        lamports: CreationLamports,
    ) -> Result<()> {
        let seeds = T::bumped_seeds(accs, ctx.program_id);
        let size = self.size();

        let s: Vec<&[u8]> = seeds.iter().map(|item| item.as_slice()).collect();
        let seed_slice = s.as_slice();

        create_account(
            ctx,
            self.info(),
            payer,
            lamports,
            size,
            &self.owner_pubkey(ctx.program_id)?,
            SignedWithSeeds(&[seed_slice]),
        )
    }
}

impl<'a, 'b: 'a, K, T: AccountSize + Seeded<K> + Keyed<'a, 'b> + MultiOwned>
    CreatableWithOwner<'a, K> for T
{
    fn create_with_owner(
        &'a self,
        owner: &Pubkey,
        accs: K,
        ctx: &'a ExecutionContext<'_, '_>,
        payer: &'a Pubkey,
        lamports: CreationLamports,
    ) -> Result<()> {
        if !self.owners_pubkeys(ctx.program_id)?.contains(owner) {
            return Err(SolitaireError::InvalidOwner(*owner));
        }

        let seeds = T::bumped_seeds(accs, ctx.program_id);
        let size = self.size();

        let s: Vec<&[u8]> = seeds.iter().map(|item| item.as_slice()).collect();
        let seed_slice = s.as_slice();

        create_account(
            ctx,
            self.info(),
            payer,
            lamports,
            size,
            owner,
            SignedWithSeeds(&[seed_slice]),
        )
    }
}

impl<'a, const SEED: &'static str, T> Seeded<Option<()>> for Derive<T, SEED> {
    fn seeds(_accs: Option<()>) -> Vec<Vec<u8>> {
        vec![SEED.as_bytes().to_vec()]
    }
}

pub fn invoke_seeded<I, T: Seeded<I>>(
    instruction: &Instruction,
    context: &ExecutionContext,
    seeded_acc: &T,
    accs: I,
) -> ProgramResult {
    let seeds = seeded_acc.self_bumped_seeds(accs, context.program_id);
    let s: Vec<&[u8]> = seeds.iter().map(|item| item.as_slice()).collect();
    let seed_slice = s.as_slice();
    invoke_signed(instruction, context.accounts, &[seed_slice])
}
