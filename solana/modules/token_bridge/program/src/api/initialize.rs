use crate::{accounts::ConfigAccount, types::*};
use solana_program::{account_info::AccountInfo, program_error::ProgramError, pubkey::Pubkey};
use solitaire::{CreationLamports::Exempt, *};
use std::ops::{Deref, DerefMut};

#[derive(FromAccounts)]
pub struct Initialize<'b> {
    pub payer: Signer<AccountInfo<'b>>,
    pub config: ConfigAccount<'b, { AccountState::Uninitialized }>,
}

impl<'b> InstructionContext<'b> for Initialize<'b> {}

pub fn initialize(
    ctx: &ExecutionContext,
    accs: &mut Initialize,
    wormhole_bridge: Pubkey,
) -> Result<()> {
    // Create the config account
    accs.config.create(ctx, accs.payer.key, Exempt)?;
    accs.config.wormhole_bridge = wormhole_bridge;
    Ok(())
}
