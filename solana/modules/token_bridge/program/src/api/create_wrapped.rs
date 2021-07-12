use crate::{
    accounts::{
        ConfigAccount, Endpoint, EndpointDerivationData, MintSigner, WrappedDerivationData,
        WrappedMint, WrappedTokenMeta,
    },
    messages::PayloadAssetMeta,
    types::*,
};
use bridge::vaa::ClaimableVAA;
use solana_program::{
    account_info::AccountInfo, program::invoke_signed, program_error::ProgramError, pubkey::Pubkey,
};
use solitaire::{processors::seeded::Seeded, CreationLamports::Exempt, *};
use spl_token::{
    error::TokenError::OwnerMismatch,
    state::{Account, Mint},
};
use std::ops::{Deref, DerefMut};

#[derive(FromAccounts)]
pub struct CreateWrapped<'b> {
    pub payer: Signer<AccountInfo<'b>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub chain_registration: Endpoint<'b, { AccountState::Initialized }>,
    pub vaa: Many<ClaimableVAA<'b, PayloadAssetMeta>>,

    // New Wrapped
    pub mint: WrappedMint<'b, { AccountState::Uninitialized }>,
    pub meta: WrappedTokenMeta<'b, { AccountState::Uninitialized }>,

    pub mint_authority: MintSigner<'b>,
}

impl<'a> From<&CreateWrapped<'a>> for EndpointDerivationData {
    fn from(accs: &CreateWrapped<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.0.meta().emitter_chain,
            emitter_address: accs.vaa.0.meta().emitter_address,
        }
    }
}

impl<'a> From<&CreateWrapped<'a>> for WrappedDerivationData {
    fn from(accs: &CreateWrapped<'a>) -> Self {
        WrappedDerivationData {
            token_chain: accs.vaa.0.token_chain,
            token_address: accs.vaa.0.token_address,
        }
    }
}

impl<'b> InstructionContext<'b> for CreateWrapped<'b> {
    fn verify(&self, program_id: &Pubkey) -> Result<()> {
        self.mint.verify_derivation(program_id, &(self.into()))?;
        self.meta.verify_derivation(program_id, &(self.into()))?;
        self.chain_registration
            .verify_derivation(program_id, &(self.into()))?;

        Ok(())
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CreateWrappedData {}

pub fn create_wrapped(
    ctx: &ExecutionContext,
    accs: &mut CreateWrapped,
    data: CreateWrappedData,
) -> Result<()> {
    accs.vaa.0.claim(ctx, accs.payer.key)?;

    // Create mint account
    accs.mint
        .create(&((&*accs).into()), ctx, accs.payer.key, Exempt);

    // Initialize mint
    let init_ix = spl_token::instruction::initialize_mint(
        &spl_token::id(),
        accs.mint.info().key,
        accs.mint_authority.key,
        None,
        8,
    )?;
    invoke_signed(&init_ix, ctx.accounts, &[])?;

    // Create meta account
    accs.meta
        .create(&((&*accs).into()), ctx, accs.payer.key, Exempt);

    // Populate meta account
    accs.meta.chain = accs.vaa.0.token_chain;
    accs.meta.token_address = accs.vaa.0.token_address;

    Ok(())
}
