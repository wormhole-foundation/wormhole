use crate::{
    constants::{
        CUSTODY_AUTHORITY_SEED_PREFIX, EMITTER_SEED_PREFIX, TRANSFER_AUTHORITY_SEED_PREFIX,
    },
    error::TokenBridgeError,
    legacy::instruction::TransferTokensWithPayloadArgs,
    utils::{self, TruncateAmount},
};
use anchor_lang::prelude::*;
use anchor_spl::token;
use core_bridge_program::sdk as core_bridge;
use ruint::aliases::U256;
use wormhole_raw_vaas::support::EncodedAmount;

#[derive(Accounts)]
pub struct TransferTokensWithPayloadNative<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// CHECK: Source token account. Because we check the mint of the custody token account, we can
    /// be sure that this token account is the same mint since the Token Program transfer
    /// instruction handler checks that the mints of these two accounts must be the same.
    #[account(mut)]
    src_token: AccountInfo<'info>,

    /// CHECK: Native mint. We ensure this mint is not one that has originated from a foreign
    /// network in access control.
    mint: Account<'info, token::Mint>,

    #[account(
        init_if_needed,
        payer = payer,
        token::mint = mint,
        token::authority = custody_authority,
        seeds = [mint.key().as_ref()],
        bump,
    )]
    custody_token: Account<'info, token::TokenAccount>,

    /// CHECK: This authority is whom the source token account owner delegates spending approval for
    /// transferring native assets or burning wrapped assets.
    #[account(
        seeds = [TRANSFER_AUTHORITY_SEED_PREFIX],
        bump
    )]
    transfer_authority: AccountInfo<'info>,

    /// CHECK: This account is the authority that can move tokens from the custody account.
    #[account(
        seeds = [CUSTODY_AUTHORITY_SEED_PREFIX],
        bump
    )]
    custody_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_message: Signer<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages.
    /// This PDA address is checked in `publish_token_bridge_message`.
    #[account(
        seeds = [EMITTER_SEED_PREFIX],
        bump
    )]
    core_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_fee_collector: Option<AccountInfo<'info>>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    /// This account is used to authorize sending the transfer with payload. This signer is either
    /// an ordinary signer (whose pubkey will be encoded as the sender address) or a program's PDA,
    /// whose seeds are [b"sender"] (and the program ID will be encoded as the sender address).
    sender_authority: Signer<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_program: Program<'info, token::Token>,
    core_bridge_program: Program<'info, core_bridge::CoreBridge>,
}

impl<'info> core_bridge::legacy::ProcessLegacyInstruction<'info, TransferTokensWithPayloadArgs>
    for TransferTokensWithPayloadNative<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyTransferTokensWithPayloadNative";

    const ANCHOR_IX_FN: fn(Context<Self>, TransferTokensWithPayloadArgs) -> Result<()> =
        transfer_tokens_with_payload_native;

    fn order_account_infos<'a>(
        account_infos: &'a [AccountInfo<'info>],
    ) -> Result<Vec<AccountInfo<'info>>> {
        super::order_transfer_tokens_with_payload_account_infos(account_infos)
    }
}

impl<'info> TransferTokensWithPayloadNative<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // Make sure the mint authority is not the Token Bridge's. If it is, then this mint
        // originated from a foreign network.
        //
        // NOTE: The assumption that the mint authority determining whether this mint originated
        // from the Token Bridge is not reliable. Someone could transfer mint authority to the Token
        // Bridge's mint authority, which would make this mint look like it belongs to the Token
        // Bridge (when in fact it was never created via the create wrapped instruction).
        if let solana_program::program_option::COption::Some(mint_authority) =
            ctx.accounts.mint.mint_authority
        {
            let (token_bridge_mint_authority, _) = Pubkey::find_program_address(
                &[crate::constants::MINT_AUTHORITY_SEED_PREFIX],
                &crate::ID,
            );
            require!(
                mint_authority != token_bridge_mint_authority,
                TokenBridgeError::WrappedAsset
            );
        }

        Ok(())
    }
}

#[access_control(TransferTokensWithPayloadNative::constraints(&ctx))]
fn transfer_tokens_with_payload_native(
    ctx: Context<TransferTokensWithPayloadNative>,
    args: TransferTokensWithPayloadArgs,
) -> Result<()> {
    let TransferTokensWithPayloadArgs {
        nonce,
        amount,
        redeemer,
        redeemer_chain,
        payload,
        cpi_program_id,
    } = args;

    // Generate sender address, which may either be the program ID or the sender authority's pubkey.
    //
    // NOTE: We perform the derivation check here instead of in the access control because we do not
    // want to spend compute units to re-derive the authority if cpi_program_id is Some(pubkey).
    let sender = crate::utils::new_sender_address(&ctx.accounts.sender_authority, cpi_program_id)?;

    let mint = &ctx.accounts.mint;

    // Deposit native assets from the source token account into the custody account.
    token::transfer(
        CpiContext::new_with_signer(
            ctx.accounts.token_program.to_account_info(),
            token::Transfer {
                from: ctx.accounts.src_token.to_account_info(),
                to: ctx.accounts.custody_token.to_account_info(),
                authority: ctx.accounts.transfer_authority.to_account_info(),
            },
            &[&[
                TRANSFER_AUTHORITY_SEED_PREFIX,
                &[ctx.bumps["transfer_authority"]],
            ]],
        ),
        mint.truncate_amount(amount),
    )?;

    // Prepare Wormhole message. We need to normalize these amounts because we are working with
    // native assets.
    let token_transfer = crate::messages::TransferWithMessage {
        norm_amount: EncodedAmount::norm(U256::from(amount), mint.decimals).0,
        token_address: ctx.accounts.mint.key().to_bytes(),
        token_chain: core_bridge::SOLANA_CHAIN,
        redeemer,
        redeemer_chain,
        sender,
        payload,
    };

    // Finally publish Wormhole message using the Core Bridge.
    utils::cpi::publish_token_bridge_message(
        CpiContext::new_with_signer(
            ctx.accounts.core_bridge_program.to_account_info(),
            core_bridge::PublishMessage {
                payer: ctx.accounts.payer.to_account_info(),
                message: ctx.accounts.core_message.to_account_info(),
                emitter_authority: ctx.accounts.core_emitter.to_account_info(),
                config: ctx.accounts.core_bridge_config.to_account_info(),
                emitter_sequence: ctx.accounts.core_emitter_sequence.to_account_info(),
                fee_collector: ctx.accounts.core_fee_collector.clone(),
                system_program: ctx.accounts.system_program.to_account_info(),
            },
            &[&[EMITTER_SEED_PREFIX, &[ctx.bumps["core_emitter"]]]],
        ),
        nonce,
        token_transfer,
    )
}
