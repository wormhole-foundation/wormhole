use super::keyed::Keyed;
use crate::{
    system_instruction,
    AccountInfo,
    AccountState,
    CreationLamports,
    Data,
    Deref,
    Derive,
    ExecutionContext,
    FromAccounts,
    Info,
    Peel,
    Result,
    Signer,
    SolitaireError,
    System,
    Sysvar,
};
use borsh::{
    BorshSchema,
    BorshSerialize,
};
use solana_program::{
    program::invoke_signed,
    pubkey::Pubkey,
};

pub trait AccountSize {
    fn size(&self) -> usize;
}

pub enum AccountOwner {
    This,
    Other(Pubkey),
    Any,
}

pub trait Owned {
    fn owner(&self) -> AccountOwner;

    fn owner_pubkey(&self, program_id: &Pubkey) -> Result<Pubkey> {
        match self.owner() {
            AccountOwner::This => Ok(*program_id),
            AccountOwner::Other(v) => Ok(v),
            AccountOwner::Any => Err(SolitaireError::AmbiguousOwner),
        }
    }
}

impl<'a, T: Owned + Default, const IsInitialized: AccountState> Owned
    for Data<'a, T, IsInitialized>
{
    fn owner(&self) -> AccountOwner {
        self.1.owner()
    }
}

pub trait Seeded<I> {
    fn seeds(&self, accs: I) -> Vec<Vec<u8>>;

    fn bumped_seeds(&self, accs: I, program_id: &Pubkey) -> Vec<Vec<u8>> {
        let mut seeds = self.seeds(accs);
        let mut s: Vec<&[u8]> = seeds.iter().map(|item| item.as_slice()).collect();
        let mut seed_slice = s.as_slice();
        let (_, bump_seed) = Pubkey::find_program_address(seed_slice, program_id);
        seeds.push(vec![bump_seed]);

        seeds
    }

    fn verify_derivation<'a, 'b: 'a>(&'a self, program_id: &'a Pubkey, accs: I) -> Result<()>
    where
        Self: Keyed<'a, 'b>,
    {
        let seeds = self.seeds(accs);
        let s: Vec<&[u8]> = seeds.iter().map(|item| item.as_slice()).collect();
        let seed_slice = s.as_slice();

        let (derived, bump) = Pubkey::find_program_address(seed_slice, program_id);
        if &derived == self.info().key {
            Ok(())
        } else {
            Err(SolitaireError::InvalidDerive(*self.info().key))
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

impl<T: BorshSerialize + Owned + Default, const IsInitialized: AccountState> AccountSize
    for Data<'_, T, IsInitialized>
{
    fn size(&self) -> usize {
        self.1.try_to_vec().unwrap().len()
    }
}

impl<'a, 'b: 'a, K, T: AccountSize + Seeded<K> + Keyed<'a, 'b> + Owned> Creatable<'a, K> for T {
    fn create(
        &'a self,
        accs: K,
        ctx: &'a ExecutionContext<'_, '_>,
        payer: &'a Pubkey,
        lamports: CreationLamports,
    ) -> Result<()> {
        let seeds = self.bumped_seeds(accs, ctx.program_id);
        let size = self.size();

        let mut s: Vec<&[u8]> = seeds.iter().map(|item| item.as_slice()).collect();
        let mut seed_slice = s.as_slice();

        let ix = system_instruction::create_account(
            payer,
            self.info().key,
            lamports.amount(size),
            size as u64,
            &self.owner_pubkey(ctx.program_id)?,
        );
        Ok(invoke_signed(&ix, ctx.accounts, &[seed_slice])?)
    }
}

impl<'a, const Seed: &'static str, T> Seeded<Option<()>> for Derive<T, Seed> {
    fn seeds(&self, accs: Option<()>) -> Vec<Vec<u8>> {
        vec![Seed.as_bytes().to_vec()]
    }
}
