use crate::{
    accounts::{
        ConfigAccount,
        CustodyAccount,
        CustodyAccountDerivationData,
        CustodySigner,
        Endpoint,
        EndpointDerivationData,
        MintSigner,
        WrappedDerivationData,
        WrappedMint,
    },
    messages::PayloadTransfer,
    types::*,
    TokenBridgeError::*,
};
use bridge::vaa::ClaimableVAA;
use solana_program::{
    account_info::AccountInfo,
    program::invoke_signed,
    program_error::ProgramError,
    pubkey::Pubkey,
};
use solitaire::{
    processors::seeded::{
        invoke_seeded,
        Seeded,
    },
    CreationLamports::Exempt,
    *,
};
use spl_token::state::{
    Account,
    Mint,
};
use std::ops::{
    Deref,
    DerefMut,
};

#[derive(FromAccounts)]
pub struct CompleteNative<'b> {
    pub payer: Signer<AccountInfo<'b>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub vaa: ClaimableVAA<'b, PayloadTransfer>,
    pub chain_registration: Endpoint<'b, { AccountState::Initialized }>,

    pub to: Data<'b, SplAccount, { AccountState::Initialized }>,
    pub custody: CustodyAccount<'b, { AccountState::Initialized }>,
    pub mint: Data<'b, SplMint, { AccountState::Initialized }>,

    pub custody_signer: CustodySigner<'b>,
}

impl<'a> From<&CompleteNative<'a>> for EndpointDerivationData {
    fn from(accs: &CompleteNative<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.meta().emitter_chain,
            emitter_address: accs.vaa.meta().emitter_address,
        }
    }
}

impl<'a> From<&CompleteNative<'a>> for CustodyAccountDerivationData {
    fn from(accs: &CompleteNative<'a>) -> Self {
        CustodyAccountDerivationData {
            mint: *accs.mint.info().key,
        }
    }
}

impl<'b> InstructionContext<'b> for CompleteNative<'b> {
    fn verify(&self, program_id: &Pubkey) -> Result<()> {
        // Verify the chain registration
        self.chain_registration
            .verify_derivation(program_id, &(self.into()))?;

        // Verify that the custody account is derived correctly
        self.custody.verify_derivation(program_id, &(self.into()))?;

        // Verify mints
        if self.mint.info().key != self.to.info().key {
            return Err(InvalidMint.into());
        }
        if self.mint.info().key != self.custody.info().key {
            return Err(InvalidMint.into());
        }
        if &self.custody.owner != self.custody_signer.key {
            return Err(InvalidMint.into());
        }

        // Verify VAA
        if self.vaa.token_address != self.mint.info().key.to_bytes() {
            return Err(InvalidMint.into());
        }
        if self.vaa.token_chain != 1 {
            return Err(InvalidChain.into());
        }

        Ok(())
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CompleteNativeData {}

pub fn complete_native(
    ctx: &ExecutionContext,
    accs: &mut CompleteNative,
    data: CompleteNativeData,
) -> Result<()> {
    // Prevent vaa double signing
    accs.vaa.claim(ctx, accs.payer.key)?;

    // Transfer tokens
    let transfer_ix = spl_token::instruction::transfer(
        &spl_token::id(),
        accs.custody.info().key,
        accs.to.info().key,
        accs.custody_signer.key,
        &[],
        accs.vaa.amount.as_u64(),
    )?;
    invoke_seeded(&transfer_ix, ctx, &accs.custody_signer, None)?;

    // TODO fee

    Ok(())
}

#[derive(FromAccounts)]
pub struct CompleteWrapped<'b> {
    pub payer: Signer<AccountInfo<'b>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    // Signed message for the transfer
    pub vaa: ClaimableVAA<'b, PayloadTransfer>,

    pub chain_registration: Endpoint<'b, { AccountState::Initialized }>,

    pub to: Data<'b, SplAccount, { AccountState::Initialized }>,
    pub mint: WrappedMint<'b, { AccountState::Initialized }>,

    pub mint_authority: MintSigner<'b>,
}

impl<'a> From<&CompleteWrapped<'a>> for EndpointDerivationData {
    fn from(accs: &CompleteWrapped<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.meta().emitter_chain,
            emitter_address: accs.vaa.meta().emitter_address,
        }
    }
}

impl<'a> From<&CompleteWrapped<'a>> for WrappedDerivationData {
    fn from(accs: &CompleteWrapped<'a>) -> Self {
        WrappedDerivationData {
            token_chain: accs.vaa.token_chain,
            token_address: accs.vaa.token_address,
        }
    }
}

impl<'b> InstructionContext<'b> for CompleteWrapped<'b> {
    fn verify(&self, program_id: &Pubkey) -> Result<()> {
        // Verify the chain registration
        self.chain_registration
            .verify_derivation(program_id, &(self.into()))?;

        // Verify mint
        self.mint.verify_derivation(program_id, &(self.into()))?;

        // Verify mints
        if self.mint.info().key != self.to.info().key {
            return Err(InvalidMint.into());
        }
        Ok(())
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CompleteWrappedData {}

pub fn complete_wrapped(
    ctx: &ExecutionContext,
    accs: &mut CompleteWrapped,
    data: CompleteWrappedData,
) -> Result<()> {
    accs.vaa.claim(ctx, accs.payer.key)?;

    // Mint tokens
    let mint_ix = spl_token::instruction::mint_to(
        &spl_token::id(),
        accs.mint.info().key,
        accs.to.info().key,
        accs.mint_authority.key,
        &[],
        accs.vaa.amount.as_u64(),
    )?;
    invoke_seeded(&mint_ix, ctx, &accs.mint_authority, None)?;

    // TODO fee

    Ok(())
}
