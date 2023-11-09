use crate::state::SignerSequence;
use anchor_lang::prelude::*;
use core_bridge_program::sdk as core_bridge;

const EMITTER_AUTHORITY_SEED_PREFIX: &[u8] = b"emitter";

#[derive(Accounts)]
#[instruction(_nonce: u32, data_len: u32)]
pub struct MockPrepareMessageV1<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        init_if_needed,
        payer = payer,
        space = 8 + SignerSequence::INIT_SPACE,
        seeds = [SignerSequence::SEED_PREFIX, payer.key().as_ref()],
        bump,
    )]
    payer_sequence: Account<'info, SignerSequence>,

    /// CHECK: Draft message.
    #[account(
        init,
        payer = payer,
        space = core_bridge::compute_prepared_message_space(data_len.try_into().unwrap()),
        seeds = [
            b"my_draft_message",
            payer.key().as_ref(),
            payer_sequence.to_le_bytes().as_ref()
        ],
        bump,
        owner = core_bridge_program::ID,
    )]
    message: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(
        seeds = [EMITTER_AUTHORITY_SEED_PREFIX],
        bump,
    )]
    emitter_authority: AccountInfo<'info>,

    core_bridge_program: Program<'info, core_bridge::CoreBridge>,
    system_program: Program<'info, System>,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct MockPrepareMessageV1Args {
    pub nonce: u32,
    pub data: Vec<u8>,
}

pub fn mock_prepare_message_v1(
    ctx: Context<MockPrepareMessageV1>,
    args: MockPrepareMessageV1Args,
) -> Result<()> {
    let MockPrepareMessageV1Args { nonce, data } = args;

    core_bridge::prepare_message(
        CpiContext::new_with_signer(
            ctx.accounts.core_bridge_program.to_account_info(),
            core_bridge::PrepareMessage {
                emitter_authority: ctx.accounts.emitter_authority.to_account_info(),
                message: ctx.accounts.message.to_account_info(),
            },
            &[&[
                EMITTER_AUTHORITY_SEED_PREFIX,
                &[ctx.bumps["emitter_authority"]],
            ]],
        ),
        core_bridge::InitMessageV1Args {
            nonce,
            cpi_program_id: Some(crate::ID),
            commitment: core_bridge::Commitment::Finalized,
        },
        data,
    )
}
