use crate::{
    accounts::ConfigAccount,
    types::{
        pauser,
        unpauser,
        write_paused,
        CONFIG_WITH_PAUSER_LEN,
    },
    TokenBridgeError::{
        InvalidPauser,
        InvalidProgramOwner,
        PauserNotConfigured,
    },
};
use solana_program::{
    account_info::AccountInfo,
    pubkey::Pubkey,
};
use solitaire::*;

/// Event discriminator: `SHA256("event:Paused")[0..8]`. Emitted via Anchor-style self-CPI
/// when `pause` succeeds. Payload: 32-byte pauser pubkey.
pub const PAUSED_EVENT_DISCRIMINATOR: [u8; 8] = [0xac, 0xf8, 0x05, 0xfd, 0x31, 0xff, 0xff, 0xe8];

/// Event discriminator: `SHA256("event:Unpaused")[0..8]`. Emitted via Anchor-style self-CPI
/// when `unpause` succeeds. Payload: 32-byte unpauser pubkey.
pub const UNPAUSED_EVENT_DISCRIMINATOR: [u8; 8] = [0x9c, 0x96, 0x2f, 0xae, 0x78, 0xd8, 0x5d, 0x75];

#[derive(FromAccounts)]
pub struct Pause<'b> {
    /// Caller must equal the configured pauser stored in the Config tail.
    pub pauser: Signer<AccountInfo<'b>>,

    pub config: Mut<ConfigAccount<'b, { AccountState::Initialized }>>,

    /// Event authority PDA for Anchor CPI event signing.
    pub event_authority: Info<'b>,

    /// This program (for self-CPI).
    pub self_program: Info<'b>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct PauseData {}

pub fn pause(ctx: &ExecutionContext, accs: &mut Pause, _data: PauseData) -> Result<()> {
    set_paused_state(
        accs.config.info(),
        accs.pauser.key,
        /* is_pauser_path */ true,
        /* paused */ true,
    )?;
    if accs.self_program.key != ctx.program_id {
        return Err(InvalidProgramOwner.into());
    }
    emit_event_cpi(
        ctx,
        &accs.event_authority,
        &PAUSED_EVENT_DISCRIMINATOR,
        &accs.pauser.key.to_bytes(),
    )
}

#[derive(FromAccounts)]
pub struct Unpause<'b> {
    /// Caller must equal the configured unpauser stored in the Config tail.
    pub unpauser: Signer<AccountInfo<'b>>,

    pub config: Mut<ConfigAccount<'b, { AccountState::Initialized }>>,

    /// Event authority PDA for Anchor CPI event signing.
    pub event_authority: Info<'b>,

    /// This program (for self-CPI).
    pub self_program: Info<'b>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct UnpauseData {}

pub fn unpause(ctx: &ExecutionContext, accs: &mut Unpause, _data: UnpauseData) -> Result<()> {
    set_paused_state(
        accs.config.info(),
        accs.unpauser.key,
        /* is_pauser_path */ false,
        /* paused */ false,
    )?;
    if accs.self_program.key != ctx.program_id {
        return Err(InvalidProgramOwner.into());
    }
    emit_event_cpi(
        ctx,
        &accs.event_authority,
        &UNPAUSED_EVENT_DISCRIMINATOR,
        &accs.unpauser.key.to_bytes(),
    )
}

fn set_paused_state(
    config_info: &AccountInfo,
    signer: &Pubkey,
    is_pauser_path: bool,
    paused: bool,
) -> Result<()> {
    // Reject if the Config account hasn't been migrated yet (legacy 32-byte layout).
    if config_info.data_len() < CONFIG_WITH_PAUSER_LEN {
        return Err(PauserNotConfigured.into());
    }

    {
        let data = config_info.data.borrow();
        let configured = if is_pauser_path {
            pauser(&data)
        } else {
            unpauser(&data)
        };
        // Per whitepapers/0003_token_bridge.md Pausing: when a role is unassigned, the entry point MUST revert
        // before comparing the caller against the configured role. The zero-pubkey can't be a
        // Signer in practice on Solana, but the spec requires the explicit unassigned-check.
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
