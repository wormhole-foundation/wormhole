use crate::accounts::ConfigAccount;
use solana_program::{
    account_info::AccountInfo,
    pubkey::Pubkey,
};
use solitaire::{
    CreationLamports::Exempt,
    *,
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
