use solana_program::{msg, pubkey::Pubkey};
use solitaire::{
    AccountState, Context, ExecutionContext, FromAccounts, Info, InstructionContext, Keyed, Peel,
    Result as SoliResult, Signer, SolitaireError, ToInstruction,
};

use crate::config::{P2WConfigAccount, Pyth2WormholeConfig};

#[derive(FromAccounts, ToInstruction)]
pub struct SetConfig<'b> {
    /// New config to apply to the program
    pub config: P2WConfigAccount<'b, { AccountState::Initialized }>,
    /// Current owner authority of the program
    pub current_owner: Signer<Info<'b>>,
    pub payer: Signer<Info<'b>>,
}

impl<'b> InstructionContext<'b> for SetConfig<'b> {
    fn deps(&self) -> Vec<Pubkey> {
        vec![]
    }
}

/// Alters the current settings of pyth2wormhole
pub fn set_config(
    _ctx: &ExecutionContext,
    accs: &mut SetConfig,
    data: Pyth2WormholeConfig,
) -> SoliResult<()> {
    if &accs.config.0.owner != accs.current_owner.info().key {
        msg!(
            "Current owner account mismatch (expected {:?})",
            accs.config.0.owner
        );
        return Err(SolitaireError::InvalidSigner(
            accs.current_owner.info().key.clone(),
        ));
    }

    accs.config.1 = data;

    Ok(())
}
