use crate::{
    accounts::ConfigAccount,
    types::*,
};
use solana_program::{
    account_info::AccountInfo,
    msg,
    program_error::ProgramError,
    pubkey::Pubkey,
};
use solitaire::{
    CreationLamports::Exempt,
    *,
};
use std::ops::{
    Deref,
    DerefMut,
};

#[derive(FromAccounts)]
pub struct Initialize<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,

    pub config: Mut<ConfigAccount<'b, { AccountState::Uninitialized }>>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct InitializeData {
    pub bridge: Pubkey,
}

impl<'b> InstructionContext<'b> for Initialize<'b> {
}

pub fn initialize(
    ctx: &ExecutionContext,
    accs: &mut Initialize,
    data: InitializeData,
) -> Result<()> {
    // Create the config account
    accs.config.create(ctx, accs.payer.key, Exempt)?;
    accs.config.wormhole_bridge = data.bridge;
    Ok(())
}
