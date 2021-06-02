use crate::{
    system_instruction,
    AccountInfo,
    CreationLamports,
    Data,
    Deref,
    Derive,
    ExecutionContext,
    FromAccounts,
    Info,
    Keyed,
    Peel,
    Result,
    Signer,
    SolitaireError,
    System,
    Sysvar,
    Uninitialized,
};
use borsh::{BorshSchema, BorshSerialize};
use solana_program::{program::invoke_signed, pubkey::Pubkey};

pub trait AccountSize {
    fn size(&self) -> usize;
}

pub trait Seeded<I> {
    fn seeds(&self, accs: I) -> Vec<Vec<Vec<u8>>>;
    fn verify_derivation(&self, program_id: &Pubkey, accs: I) -> Result<()>
    where
        Self: Keyed,
    {
        let seeds = self.seeds(accs);
        let (derived, bump) = Pubkey::find_program_address(&[], program_id); //TODO
        if &derived == self.pubkey() {
            Ok(())
        } else {
            Err(SolitaireError::InvalidDerive(*self.pubkey()))
        }
    }
}

pub trait Creatable<I> {
    fn create(
        &self,
        accs: I,
        ctx: &ExecutionContext,
        payer: &Pubkey,
        lamports: CreationLamports,
    ) -> Result<()>;
}

impl<T: BorshSerialize, const IsInitialized: bool> AccountSize for Data<'_, T, IsInitialized> {
    fn size(&self) -> usize {
        self.1.try_to_vec().unwrap().len()
    }
}

impl<'a, 'b: 'a, K, T: AccountSize + Seeded<K> + Keyed> Creatable<K> for T {
    fn create(
        &self,
        accs: K,
        ctx: &ExecutionContext<'_, '_>,
        payer: &Pubkey,
        lamports: CreationLamports,
    ) -> Result<()> {
        let seeds = self.seeds(accs);
        let size = self.size();

        let ix = system_instruction::create_account(
            payer,
            self.pubkey(),
            lamports.amount(size),
            size as u64,
            ctx.program_id,
        );
        Ok(invoke_signed(&ix, ctx.accounts, &[])?) // TODO use seeds
    }
}

impl<const Seed: &'static str, T: BorshSerialize> Creatable<Option<()>>
    for Derive<Data<'_, T, Uninitialized>, Seed>
{
    fn create(
        &self,
        _: Option<()>,
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
        Ok(invoke_signed(&ix, ctx.accounts, &[&[Seed.as_bytes()]])?)
    }
}
