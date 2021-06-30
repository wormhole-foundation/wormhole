use solana_program::pubkey::Pubkey;
use solitaire::{
    AccountState, Context, Creatable, CreationLamports, ExecutionContext, FromAccounts, Info,
    InstructionContext, Keyed, Peel, Result as SoliResult, Signer, ToInstruction,
};

use crate::config::{P2WConfigAccount, Pyth2WormholeConfig};

#[derive(FromAccounts, ToInstruction)]
pub struct Initialize<'b> {
    pub new_config: P2WConfigAccount<'b, {AccountState::Uninitialized}>,
    pub payer: Signer<Info<'b>>,
}

impl<'b> InstructionContext<'b> for Initialize<'b> {
    fn verify(&self, _program_id: &Pubkey) -> SoliResult<()> {
        Ok(())
    }

    fn deps(&self) -> Vec<Pubkey> {
        vec![]
    }
}

/// Must be called right after deployment
pub fn initialize(
    ctx: &ExecutionContext,
    accs: &mut Initialize,
    data: Pyth2WormholeConfig,
) -> SoliResult<()> {
    accs.new_config
        .create(ctx, accs.payer.info().key, CreationLamports::Exempt)?;
    accs.new_config.1 = data;

    Ok(())
}
