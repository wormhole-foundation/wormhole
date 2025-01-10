use anchor_lang::prelude::*;
use borsh::{BorshDeserialize, BorshSerialize};
use wormhole_anchor_sdk::wormhole::{self, Finality};
use wormhole_solana_consts::{
    CORE_BRIDGE_CONFIG, CORE_BRIDGE_FEE_COLLECTOR, CORE_BRIDGE_PROGRAM_ID,
};

declare_id!("EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX");

#[program]
pub mod wormhole_post_message_shim {
    use super::*;
    use anchor_lang::solana_program;

    /// This instruction is intended to be a significantly cheaper alternative to `post_message` on the core bridge.
    /// It achieves this by reusing the message account, per emitter, via `post_message_unreliable` and
    /// emitting a CPI event for the guardian to observe containing the information previously only found
    /// in the resulting message account. Since this passes through the emitter and calls `post_message_unreliable`
    /// on the core bridge, it can be used (or not used) without disruption.
    ///
    /// NOTE: In the initial message publication for a new emitter, this will require one additional CPI call depth
    /// when compared to using the core bridge directly. If that is an issue, simply emit an empty message on initialization
    /// (or migration) in order to instantiate the account. This will result in a VAA from your emitter, so be careful to
    /// avoid any issues that may result in.
    ///
    /// Direct case
    /// shim `PostMessage` -> core `0x8`
    ///                    -> shim `MesssageEvent`
    ///
    /// Integration case
    /// Integrator Program -> shim `PostMessage` -> core `0x8`
    ///                                          -> shim `MesssageEvent`
    pub fn post_message(
        ctx: Context<PostMessage>,
        nonce: u32,
        consistency_level: Finality,
        _payload: Vec<u8>,
    ) -> Result<()> {
        let ix = solana_program::instruction::Instruction {
            program_id: ctx.accounts.wormhole_program.key(),
            accounts: vec![
                AccountMeta::new(ctx.accounts.bridge.key(), false),
                AccountMeta::new(ctx.accounts.message.key(), true),
                AccountMeta::new_readonly(ctx.accounts.emitter.key(), true),
                AccountMeta::new(ctx.accounts.sequence.key(), false),
                AccountMeta::new(ctx.accounts.payer.key(), true),
                AccountMeta::new(ctx.accounts.fee_collector.key(), false),
                AccountMeta::new_readonly(ctx.accounts.clock.key(), false),
                AccountMeta::new_readonly(ctx.accounts.system_program.key(), false),
                AccountMeta::new_readonly(ctx.accounts.rent.key(), false),
            ],
            data: Instruction::PostMessageUnreliable {
                nonce,
                payload: vec![],
                consistency_level,
            }
            .try_to_vec()?,
        };
        solana_program::program::invoke_signed(
            &ix,
            &[
                // TODO: it may be possible to omit some of these
                ctx.accounts.bridge.to_account_info(),
                ctx.accounts.message.to_account_info(),
                ctx.accounts.emitter.to_account_info(),
                ctx.accounts.sequence.to_account_info(),
                ctx.accounts.payer.to_account_info(),
                ctx.accounts.fee_collector.to_account_info(),
                ctx.accounts.clock.to_account_info(),
                ctx.accounts.rent.to_account_info(),
                ctx.accounts.system_program.to_account_info(),
                ctx.accounts.wormhole_program.to_account_info(),
            ],
            &[&[&ctx.accounts.emitter.key.to_bytes(), &[ctx.bumps.message]]],
        )?;
        // parse the sequence from the account and emit the event
        // reading the account after avoids having to handle when the account doesn't exist
        let mut buf = &ctx.accounts.sequence.try_borrow_mut_data()?[..];
        let seq = wormhole::SequenceTracker::try_deserialize(&mut buf)?;
        emit_cpi!(MessageEvent {
            emitter: ctx.accounts.emitter.key(),
            sequence: seq.sequence - 1, // the sequence was incremented after the post
            submission_time: Clock::get()?.unix_timestamp as u32, // this is the same casting that the core bridge performs in post_message_internal
        });
        Ok(())
    }
}

#[event]
pub struct MessageEvent {
    emitter: Pubkey,
    sequence: u64,
    submission_time: u32,
}

#[event_cpi]
#[derive(Accounts)]
/// The accounts are ordered and named the same as the core bridge's post_message_unreliable instruction
/// TODO: some of these checks were included for IDL generation / convenience but are completely unnecessary
/// and costly on-chain. Use configuration to generate the nice IDL but omit the checks on-chain except for
/// the wormhole program. Alternatively, make this program without Anchor at all.
/// some comparison of compute units consumed:
/// - core post_message:                      25097
/// - shim without sysvar and address checks: 45608 (20511 more)
/// - shim with sysvar and address checks:    45782 (  174 more)
pub struct PostMessage<'info> {
    #[account(mut, address = CORE_BRIDGE_CONFIG)]
    /// CHECK: Wormhole bridge config. [`wormhole::post_message`] requires this account be mutable.
    pub bridge: UncheckedAccount<'info>,

    #[account(mut, seeds = [&emitter.key.to_bytes()], bump)]
    /// CHECK: Wormhole Message. [`wormhole::post_message`] requires this account be signer and mutable.
    /// This program uses a PDA per emitter, since these are already bottle-necked by sequence and
    /// the bridge enforces that emitter must be identical for reused accounts.
    /// While this could be managed by the integrator, it seems more effective to have the shim manage these accounts.
    /// Bonus, this also allows Anchor to automatically handle deriving the address.
    pub message: UncheckedAccount<'info>,

    /// CHECK: Emitter of the VAA. [`wormhole::post_message`] requires this account be signer.
    pub emitter: Signer<'info>,

    #[account(mut)]
    /// CHECK: Emitter's sequence account. [`wormhole::post_message`] requires this account be mutable.
    /// Explicitly do not re-derive this account. The core bridge verifies the derivation anyway and
    /// as of Anchor 0.30.1, auto-derivation for other programs' accounts via IDL doesn't work.
    pub sequence: UncheckedAccount<'info>,

    #[account(mut)]
    /// Payer will pay Wormhole fee to post a message.
    pub payer: Signer<'info>,

    #[account(mut, address = CORE_BRIDGE_FEE_COLLECTOR)]
    /// CHECK: Wormhole fee collector. [`wormhole::post_message`] requires this account be mutable.
    pub fee_collector: UncheckedAccount<'info>,

    /// Clock sysvar.
    pub clock: Sysvar<'info, Clock>,

    /// System program.
    pub system_program: Program<'info, System>,

    /// Rent sysvar.
    pub rent: Sysvar<'info, Rent>,

    #[account(address = CORE_BRIDGE_PROGRAM_ID)]
    /// CHECK: Wormhole program.
    pub wormhole_program: UncheckedAccount<'info>,
}

// Adapted from wormhole-anchor-sdk instructions.rs
#[derive(AnchorDeserialize, AnchorSerialize)]
/// Wormhole instructions.
pub enum Instruction {
    Initialize,
    PostMessage,
    PostVAA,
    SetFees,
    TransferFees,
    UpgradeContract,
    UpgradeGuardianSet,
    VerifySignatures,
    PostMessageUnreliable {
        nonce: u32,
        payload: Vec<u8>,
        consistency_level: Finality,
    },
}
