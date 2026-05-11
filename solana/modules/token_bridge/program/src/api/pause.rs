use crate::{
    accounts::ConfigAccount,
    types::{
        read_pauser,
        read_unpauser,
        write_paused,
        CONFIG_FULL_LEN,
    },
    TokenBridgeError::{
        InvalidPauser,
        PauserNotConfigured,
    },
};
use solana_program::{
    account_info::AccountInfo,
    pubkey::Pubkey,
};
use solitaire::*;

#[derive(FromAccounts)]
pub struct Pause<'b> {
    /// Caller must equal the configured pauser stored in the Config tail.
    pub pauser: Signer<AccountInfo<'b>>,

    pub config: Mut<ConfigAccount<'b, { AccountState::Initialized }>>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct PauseData {}

pub fn pause(_ctx: &ExecutionContext, accs: &mut Pause, _data: PauseData) -> Result<()> {
    set_paused_state(
        accs.config.info(),
        accs.pauser.key,
        /* is_pauser_path */ true,
        /* paused */ true,
    )
}

#[derive(FromAccounts)]
pub struct Unpause<'b> {
    /// Caller must equal the configured unpauser stored in the Config tail.
    pub unpauser: Signer<AccountInfo<'b>>,

    pub config: Mut<ConfigAccount<'b, { AccountState::Initialized }>>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct UnpauseData {}

pub fn unpause(_ctx: &ExecutionContext, accs: &mut Unpause, _data: UnpauseData) -> Result<()> {
    set_paused_state(
        accs.config.info(),
        accs.unpauser.key,
        /* is_pauser_path */ false,
        /* paused */ false,
    )
}

fn set_paused_state(
    config_info: &AccountInfo,
    signer: &Pubkey,
    is_pauser_path: bool,
    paused: bool,
) -> Result<()> {
    // Reject if the Config account hasn't been migrated yet (legacy 32-byte layout).
    if config_info.data_len() < CONFIG_FULL_LEN {
        return Err(PauserNotConfigured.into());
    }

    {
        let data = config_info.data.borrow();
        let configured = if is_pauser_path {
            read_pauser(&data)
        } else {
            read_unpauser(&data)
        };
        if configured == Pubkey::default() {
            return Err(PauserNotConfigured.into());
        }
        if &configured != signer {
            return Err(InvalidPauser.into());
        }
    }

    let mut data = config_info.data.borrow_mut();
    write_paused(&mut data, paused);
    Ok(())
}
