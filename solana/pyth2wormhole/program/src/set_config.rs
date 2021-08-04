use solana_program::{
    msg,
    pubkey::Pubkey,
};
use solitaire::{
    AccountState,
    ExecutionContext,
    FromAccounts,
    Info,
    InstructionContext,
    Keyed,
    Mut,
    Peel,
    Result as SoliResult,
    Signer,
    SolitaireError,
    ToInstruction,
};

use crate::config::{
    P2WConfigAccount,
    Pyth2WormholeConfig,
};

#[derive(FromAccounts, ToInstruction)]
pub struct SetConfig<'b> {
    /// Current config used by the program
    pub config: Mut<P2WConfigAccount<'b, { AccountState::Initialized }>>,
    /// Current owner authority of the program
    pub current_owner: Mut<Signer<Info<'b>>>,
    /// Payer account for updating the account data
    pub payer: Mut<Signer<Info<'b>>>,
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
