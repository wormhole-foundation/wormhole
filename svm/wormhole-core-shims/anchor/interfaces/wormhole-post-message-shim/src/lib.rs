use anchor_lang::prelude::*;
use wormhole_svm_definitions::{CORE_BRIDGE_PROGRAM_ID, POST_MESSAGE_SHIM_PROGRAM_ID};

declare_id!(POST_MESSAGE_SHIM_PROGRAM_ID);

#[program]
pub mod wormhole_post_message_shim {
    use super::*;

    /// This instruction is intended to be a significantly cheaper alternative
    /// to `post_message` on the core bridge. It achieves this by reusing the
    /// message account, per emitter, via `post_message_unreliable` and emitting
    /// a CPI event for the guardian to observe containing the information
    /// previously only found in the resulting message account. Since this
    /// passes through the emitter and calls `post_message_unreliable` on the
    /// core bridge, it can be used (or not used) without disruption.
    ///
    /// NOTE: In the initial message publication for a new emitter, this will
    /// require one additional CPI call depth when compared to using the core
    /// bridge directly. If that is an issue, simply emit an empty message on
    /// initialization (or migration) in order to instantiate the account. This
    /// will result in a VAA from your emitter, so be careful to avoid any
    /// issues that may result in.
    ///
    /// Direct case
    /// shim `PostMessage` -> core `0x8`
    ///                    -> shim `MesssageEvent`
    ///
    /// Integration case
    /// Integrator Program -> shim `PostMessage` -> core `0x8`
    ///                                          -> shim `MesssageEvent`
    pub fn post_message(
        _ctx: Context<PostMessage>,
        nonce: u32,
        consistency_level: Finality,
        payload: Vec<u8>,
    ) -> Result<()> {
        let _ = (nonce, consistency_level, payload);
        err!(ErrorCode::InstructionMissing)
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, AnchorDeserialize, AnchorSerialize)]
pub enum Finality {
    Confirmed,
    Finalized,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
#[event]
pub struct MessageEvent {
    pub emitter: Pubkey,
    pub sequence: u64,
    pub submission_time: u32,
}

/// The accounts are ordered and named the same as the core bridge's
/// post_message_unreliable instruction.
///
/// TODO: some of these checks were included for IDL generation / convenience
/// but are completely unnecessary and costly on-chain. Use configuration to
/// generate the nice IDL but omit the checks on-chain except for the wormhole
/// program. Alternatively, make this program without Anchor at all.
///
/// Some comparison of compute units consumed:
/// - core post_message:                      25097
/// - shim without sysvar and address checks: 45608 (20511 more)
/// - shim with sysvar and address checks:    45782 (  174 more)
#[derive(Accounts)]
#[event_cpi]
pub struct PostMessage<'info> {
    /// CHECK: Wormhole bridge config. [`wormhole::post_message`] requires this
    /// account be mutable.
    #[account(
        mut,
        seeds = [b"Bridge"],
        bump,
        seeds::program = CORE_BRIDGE_PROGRAM_ID
    )]
    bridge: UncheckedAccount<'info>,

    /// CHECK: Wormhole Message. [`wormhole::post_message`] requires this
    /// account be signer and mutable.
    ///
    /// This program uses a PDA per emitter, since these are already
    /// bottle-necked by sequence and the bridge enforces that emitter must be
    /// identical for reused accounts. While this could be managed by the
    /// integrator, it seems more effective to have the shim manage these
    /// accounts.
    ///
    /// Bonus, this also allows Anchor to automatically handle deriving the
    /// address.
    #[account(
        mut,
        seeds = [&emitter.key.to_bytes()],
        bump
    )]
    message: UncheckedAccount<'info>,

    /// CHECK: Emitter of the VAA. [`wormhole::post_message`] requires this
    /// account be signer.
    emitter: Signer<'info>,

    /// CHECK: Emitter's sequence account. [`wormhole::post_message`] requires
    /// this account be mutable.
    ///
    /// Explicitly do not re-derive this account. The core bridge verifies the
    /// derivation anyway and as of Anchor 0.30.1, auto-derivation for other
    /// programs' accounts via IDL doesn't work.
    #[account(
        mut,
        seeds = [b"Sequence", &emitter.key.to_bytes()],
        bump,
        seeds::program = CORE_BRIDGE_PROGRAM_ID
    )]
    sequence: UncheckedAccount<'info>,

    /// Payer will pay the rent for the Wormhole Core Bridge emitter sequence
    /// and message on the first post message call. Subsequent calls will not
    /// require more lamports for rent.
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Wormhole fee collector. [`wormhole::post_message`] requires this
    /// account be mutable.
    #[account(
        mut,
        seeds = [b"fee_collector"],
        bump,
        seeds::program = CORE_BRIDGE_PROGRAM_ID
    )]
    fee_collector: UncheckedAccount<'info>,

    /// Clock sysvar.
    clock: Sysvar<'info, Clock>,

    /// System program.
    system_program: Program<'info, System>,

    /// Rent sysvar.
    rent: Sysvar<'info, Rent>,

    #[account(address = CORE_BRIDGE_PROGRAM_ID)]
    /// CHECK: Wormhole program.
    wormhole_program: UncheckedAccount<'info>,
}
