use crate::{
    accounts::ConfigAccount,
    types::{
        freezer,
        pause_expiry,
        paused,
        pauser,
        unpauser,
        write_pause_expiry,
        write_paused,
        CONFIG_WITH_PAUSER_LEN,
    },
    TokenBridgeError::{
        InvalidPauser,
        InvalidSelfProgram,
        NotExpired,
        NotFreezer,
        NotPaused,
        PauserNotConfigured,
    },
};
use solana_program::{
    account_info::AccountInfo,
    pubkey::Pubkey,
    sysvar::clock::Clock,
};
use solitaire::*;

/// Temporary-pause duration: 5 days, in seconds (`Clock.unix_timestamp` is seconds). A `pause`
/// holds the bridge for this long; the pauser must re-`pause` to extend it (a dead-man's switch —
/// the hold lapses if the pauser stops acting). See whitepapers/0003_token_bridge.md.
pub const PAUSE_DURATION: i64 = 5 * 24 * 60 * 60;

/// Event discriminator: `SHA256("event:Paused")[0..8]`. Emitted via Anchor-style self-CPI
/// when `pause` succeeds. Payload: 32-byte pauser pubkey followed by the 8-byte LE `pause_expiry`.
pub const PAUSED_EVENT_DISCRIMINATOR: [u8; 8] = [0xac, 0xf8, 0x05, 0xfd, 0x31, 0xff, 0xff, 0xe8];

/// Event discriminator: `SHA256("event:Frozen")[0..8]`. Emitted via Anchor-style self-CPI
/// when `freeze` succeeds. Payload: 32-byte freezer pubkey followed by the 8-byte LE `pause_expiry`
/// (the maximum timestamp).
pub const FROZEN_EVENT_DISCRIMINATOR: [u8; 8] = [0x73, 0x4d, 0xbd, 0x53, 0x51, 0x47, 0xf5, 0xe8];

/// Event discriminator: `SHA256("event:Unpaused")[0..8]`. Emitted via Anchor-style self-CPI
/// when `unpause` succeeds. Payload: 32-byte unpauser pubkey.
pub const UNPAUSED_EVENT_DISCRIMINATOR: [u8; 8] = [0x9c, 0x96, 0x2f, 0xae, 0x78, 0xd8, 0x5d, 0x75];

/// Event discriminator: `SHA256("event:UnpauseExpired")[0..8]`. Emitted via Anchor-style self-CPI
/// when the permissionless `unpause_expired` succeeds. Payload: 32-byte caller pubkey.
pub const UNPAUSE_EXPIRED_EVENT_DISCRIMINATOR: [u8; 8] =
    [0x9a, 0x44, 0xa1, 0x6f, 0x91, 0xf0, 0xf3, 0x07];

#[derive(FromAccounts)]
pub struct Pause<'b> {
    /// Caller must equal the configured pauser stored in the Config tail.
    pub pauser: Signer<AccountInfo<'b>>,

    pub config: Mut<ConfigAccount<'b, { AccountState::Initialized }>>,

    /// Clock sysvar, for the pause-expiry timestamp.
    pub clock: Sysvar<'b, Clock>,

    /// Event authority PDA for Anchor CPI event signing.
    pub event_authority: Info<'b>,

    /// This program (for self-CPI).
    pub self_program: Info<'b>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct PauseData {}

/// Temporarily pause the bridge. Only callable by the configured pauser. Sets `paused` and pushes
/// `pause_expiry` to `now + PAUSE_DURATION` (5 days), but NEVER reduces an expiry already further
/// in the future — so a lower-trust pauser cannot curtail a `freeze`. Not idempotent: each call
/// extends the window.
pub fn pause(ctx: &ExecutionContext, accs: &mut Pause, _data: PauseData) -> Result<()> {
    let config_info = accs.config.info();
    require_role(config_info, accs.pauser.key, Role::Pauser)?;

    let new_expiry = accs.clock.unix_timestamp.saturating_add(PAUSE_DURATION);
    {
        let mut data = config_info.data.borrow_mut();
        // Never reduce an expiry already further out (e.g. one set by `freeze`).
        if new_expiry > pause_expiry(&data) {
            write_pause_expiry(&mut data, new_expiry);
        }
        write_paused(&mut data, true);
    }

    if accs.self_program.key != ctx.program_id {
        return Err(InvalidSelfProgram.into());
    }
    let expiry = pause_expiry(&config_info.data.borrow());
    let mut payload = [0u8; 40];
    payload[..32].copy_from_slice(&accs.pauser.key.to_bytes());
    payload[32..].copy_from_slice(&expiry.to_le_bytes());
    emit_event_cpi(
        ctx,
        &accs.event_authority,
        &PAUSED_EVENT_DISCRIMINATOR,
        &payload,
    )
}

#[derive(FromAccounts)]
pub struct Freeze<'b> {
    /// Caller must equal the configured freezer stored in the Config tail.
    pub freezer: Signer<AccountInfo<'b>>,

    pub config: Mut<ConfigAccount<'b, { AccountState::Initialized }>>,

    /// Event authority PDA for Anchor CPI event signing.
    pub event_authority: Info<'b>,

    /// This program (for self-CPI).
    pub self_program: Info<'b>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct FreezeData {}

/// Freeze the bridge for the maximum duration. Only callable by the configured freezer. Sets
/// `paused` and `pause_expiry` to the maximum timestamp. The higher-trust counterpart to the
/// temporary, self-expiring `pause`: a frozen bridge will not become permissionlessly unpausable
/// in practice and can only be lifted by the `unpauser`. Idempotent. Takes no Clock — it assigns a
/// constant expiry.
pub fn freeze(ctx: &ExecutionContext, accs: &mut Freeze, _data: FreezeData) -> Result<()> {
    let config_info = accs.config.info();
    require_role(config_info, accs.freezer.key, Role::Freezer)?;

    {
        let mut data = config_info.data.borrow_mut();
        write_pause_expiry(&mut data, i64::MAX);
        write_paused(&mut data, true);
    }

    if accs.self_program.key != ctx.program_id {
        return Err(InvalidSelfProgram.into());
    }
    let mut payload = [0u8; 40];
    payload[..32].copy_from_slice(&accs.freezer.key.to_bytes());
    payload[32..].copy_from_slice(&i64::MAX.to_le_bytes());
    emit_event_cpi(
        ctx,
        &accs.event_authority,
        &FROZEN_EVENT_DISCRIMINATOR,
        &payload,
    )
}

#[derive(FromAccounts)]
pub struct Unpause<'b> {
    /// Caller must equal the configured unpauser stored in the Config tail.
    pub unpauser: Signer<AccountInfo<'b>>,

    pub config: Mut<ConfigAccount<'b, { AccountState::Initialized }>>,

    /// Clock sysvar, for recording the unpause timestamp into `pause_expiry`.
    pub clock: Sysvar<'b, Clock>,

    /// Event authority PDA for Anchor CPI event signing.
    pub event_authority: Info<'b>,

    /// This program (for self-CPI).
    pub self_program: Info<'b>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct UnpauseData {}

/// Unpause the bridge. Only callable by the configured unpauser. Clears `paused` and sets
/// `pause_expiry` to `now`. The privileged path to lift any pause (including a `freeze`) early.
/// Reverts if the unpauser is unassigned or the bridge is not currently paused.
///
/// Recording `now` (rather than 0) leaves on-chain evidence of the last unpause while bringing any
/// stale `freeze` expiry down to the present so it cannot block a later `pause`.
pub fn unpause(ctx: &ExecutionContext, accs: &mut Unpause, _data: UnpauseData) -> Result<()> {
    let config_info = accs.config.info();
    require_role(config_info, accs.unpauser.key, Role::Unpauser)?;

    {
        let mut data = config_info.data.borrow_mut();
        if !paused(&data) {
            return Err(NotPaused.into());
        }
        write_pause_expiry(&mut data, accs.clock.unix_timestamp);
        write_paused(&mut data, false);
    }

    if accs.self_program.key != ctx.program_id {
        return Err(InvalidSelfProgram.into());
    }
    emit_event_cpi(
        ctx,
        &accs.event_authority,
        &UNPAUSED_EVENT_DISCRIMINATOR,
        &accs.unpauser.key.to_bytes(),
    )
}

#[derive(FromAccounts)]
pub struct UnpauseExpired<'b> {
    /// Pays transaction fees only — this entry point is permissionless and carries no role.
    pub payer: Mut<Signer<AccountInfo<'b>>>,

    pub config: Mut<ConfigAccount<'b, { AccountState::Initialized }>>,

    /// Clock sysvar, to check the pause has expired.
    pub clock: Sysvar<'b, Clock>,

    /// Event authority PDA for Anchor CPI event signing.
    pub event_authority: Info<'b>,

    /// This program (for self-CPI).
    pub self_program: Info<'b>,
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct UnpauseExpiredData {}

/// Permissionlessly unpause the bridge once its pause has expired. Clears `paused` and sets
/// `pause_expiry` to `now`. No role required (any fee payer may call). Reverts if the bridge is
/// not currently paused or `now < pause_expiry`.
///
/// Bounds a `pauser`-initiated pause to `PAUSE_DURATION` without requiring the `unpauser` to act.
/// The boolean `paused` remains authoritative — a pause is only lifted by an explicit
/// `unpause`/`unpause_expired` call, never silently by the passage of time.
pub fn unpause_expired(
    ctx: &ExecutionContext,
    accs: &mut UnpauseExpired,
    _data: UnpauseExpiredData,
) -> Result<()> {
    let config_info = accs.config.info();

    // Migration check: a legacy (un-extended) Config can never be paused, but guard explicitly.
    if config_info.data_len() < CONFIG_WITH_PAUSER_LEN {
        return Err(NotPaused.into());
    }

    {
        let mut data = config_info.data.borrow_mut();
        if !paused(&data) {
            return Err(NotPaused.into());
        }
        if accs.clock.unix_timestamp < pause_expiry(&data) {
            return Err(NotExpired.into());
        }
        write_pause_expiry(&mut data, accs.clock.unix_timestamp);
        write_paused(&mut data, false);
    }

    if accs.self_program.key != ctx.program_id {
        return Err(InvalidSelfProgram.into());
    }
    emit_event_cpi(
        ctx,
        &accs.event_authority,
        &UNPAUSE_EXPIRED_EVENT_DISCRIMINATOR,
        &accs.payer.key.to_bytes(),
    )
}

/// Which pause-authority role an entry point is gated on.
enum Role {
    Pauser,
    Freezer,
    Unpauser,
}

/// Verify `signer` is the configured holder of `role`. Reverts before comparing the caller if the
/// role is unassigned (zero pubkey) or the Config account has not been migrated — so an all-zero
/// role is never authorized (whitepapers/0003_token_bridge.md Pausing). Reads only; writes happen
/// in the caller after the borrow is released.
fn require_role(config_info: &AccountInfo, signer: &Pubkey, role: Role) -> Result<()> {
    // Reject if the Config account hasn't been migrated yet (legacy layout → role unassigned).
    if config_info.data_len() < CONFIG_WITH_PAUSER_LEN {
        return match role {
            Role::Freezer => Err(NotFreezer.into()),
            Role::Pauser | Role::Unpauser => Err(PauserNotConfigured.into()),
        };
    }

    let data = config_info.data.borrow();
    let configured = match role {
        Role::Pauser => pauser(&data),
        Role::Freezer => freezer(&data),
        Role::Unpauser => unpauser(&data),
    };

    // Unassigned-role check first (the zero pubkey can't be a Signer in practice, but the spec
    // requires the explicit check before comparing the caller).
    if configured == Pubkey::default() {
        return match role {
            Role::Freezer => Err(NotFreezer.into()),
            Role::Pauser | Role::Unpauser => Err(PauserNotConfigured.into()),
        };
    }
    if &configured != signer {
        return match role {
            Role::Freezer => Err(NotFreezer.into()),
            Role::Pauser | Role::Unpauser => Err(InvalidPauser.into()),
        };
    }
    Ok(())
}
