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

    pub fn post_message(ctx: Context<PostMessage>, data: PostMessageData) -> Result<()> {
        let signer_seeds: &[&[&[u8]]] =
            &[&[&ctx.accounts.emitter.key.to_bytes(), &[ctx.bumps.message]]];
        let cpi = CpiContext::new_with_signer(
            ctx.accounts.wormhole_program.to_account_info(),
            wormhole::PostMessage {
                config: ctx.accounts.bridge.to_account_info(),
                message: ctx.accounts.message.to_account_info(),
                emitter: ctx.accounts.emitter.to_account_info(),
                sequence: ctx.accounts.sequence.to_account_info(),
                payer: ctx.accounts.payer.to_account_info(),
                fee_collector: ctx.accounts.fee_collector.to_account_info(),
                clock: ctx.accounts.clock.to_account_info(),
                rent: ctx.accounts.rent.to_account_info(),
                system_program: ctx.accounts.system_program.to_account_info(),
            },
            signer_seeds,
        );
        let ix = solana_program::instruction::Instruction {
            program_id: cpi.program.key(),
            accounts: vec![
                AccountMeta::new(cpi.accounts.config.key(), false),
                AccountMeta::new(cpi.accounts.message.key(), true),
                AccountMeta::new_readonly(cpi.accounts.emitter.key(), true),
                AccountMeta::new(cpi.accounts.sequence.key(), false),
                AccountMeta::new(cpi.accounts.payer.key(), true),
                AccountMeta::new(cpi.accounts.fee_collector.key(), false),
                AccountMeta::new_readonly(cpi.accounts.clock.key(), false),
                AccountMeta::new_readonly(cpi.accounts.system_program.key(), false),
                AccountMeta::new_readonly(cpi.accounts.rent.key(), false),
            ],
            data: Instruction::PostMessageUnreliable {
                nonce: data.nonce,
                payload: vec![],
                consistency_level: data.consistency_level,
            }
            .try_to_vec()?,
        };
        solana_program::program::invoke_signed(
            &ix,
            &ToAccountInfos::to_account_infos(&cpi),
            cpi.signer_seeds,
        )?;
        // if the post was successful, parse the sequence from the account and emit the event
        let mut buf = &ctx.accounts.sequence.try_borrow_mut_data()?[..];
        let seq = wormhole::SequenceTracker::try_deserialize(&mut buf)?;
        emit_cpi!(MessageEvent {
            emitter: ctx.accounts.emitter.key(),
            sequence: seq.sequence - 1,
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

#[derive(AnchorDeserialize, AnchorSerialize)]
pub struct PostMessageData {
    /// Unique nonce for this message
    pub nonce: u32,

    /// Message payload
    pub payload: Vec<u8>,

    /// Commitment Level required for an attestation to be produced
    pub consistency_level: Finality,
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
