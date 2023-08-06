use crate::{
    constants::{
        CUSTODY_AUTHORITY_SEED_PREFIX, EMITTER_SEED_PREFIX, TRANSFER_AUTHORITY_SEED_PREFIX,
    },
    legacy::TransferTokensWithPayloadArgs,
    processor::{deposit_native_tokens, post_token_bridge_message},
    zero_copy::Mint,
};
use anchor_lang::prelude::*;
use anchor_spl::token;
use core_bridge_program::sdk as core_bridge_sdk;
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
    mint: AccountInfo<'info>,

    #[account(
        init_if_needed,
        payer = payer,
        token::mint = mint,
        token::authority = custody_authority,
        seeds = [mint.key().as_ref()],
        bump,
    )]
    custody_token: Box<Account<'info, token::TokenAccount>>,

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
    #[account(mut)]
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_message: Signer<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages.
    #[account(
        seeds = [EMITTER_SEED_PREFIX],
        bump,
    )]
    core_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_fee_collector: Option<UncheckedAccount<'info>>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    /// This account is used to authorize sending the transfer with payload. This signer is either
    /// an ordinary signer (whose pubkey will be encoded as the sender address) or a program's PDA,
    /// whose seeds are [b"sender"] (and the program ID will be encoded as the sender address).
    sender_authority: Signer<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, core_bridge_sdk::cpi::CoreBridge>,
    token_program: Program<'info, token::Token>,
}

impl<'info>
    core_bridge_program::legacy::utils::ProcessLegacyInstruction<
        'info,
        TransferTokensWithPayloadArgs,
    > for TransferTokensWithPayloadNative<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyTransferTokensWithPayloadNative";

    const ANCHOR_IX_FN: fn(Context<Self>, TransferTokensWithPayloadArgs) -> Result<()> =
        transfer_tokens_with_payload_native;
}

impl<'info> core_bridge_sdk::cpi::InvokeCoreBridge<'info>
    for TransferTokensWithPayloadNative<'info>
{
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::CreateAccount<'info> for TransferTokensWithPayloadNative<'info> {
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::PublishMessage<'info> for TransferTokensWithPayloadNative<'info> {
    fn core_bridge_config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn core_message(&self) -> AccountInfo<'info> {
        self.core_message.to_account_info()
    }

    fn core_emitter(&self) -> Option<AccountInfo<'info>> {
        Some(self.core_emitter.to_account_info())
    }

    fn core_emitter_sequence(&self) -> AccountInfo<'info> {
        self.core_emitter_sequence.to_account_info()
    }

    fn core_fee_collector(&self) -> Option<AccountInfo<'info>> {
        self.core_fee_collector
            .as_ref()
            .map(|acc| acc.to_account_info())
    }
}

impl<'info> TransferTokensWithPayloadNative<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // Make sure the mint authority is not the Token Bridge's. If it is, then this mint
        // originated from a foreign network.
        crate::utils::require_native_mint(&ctx.accounts.mint)
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

    // Deposit native assets from the source token account into the custody account.
    let amount = deposit_native_tokens(
        &ctx.accounts.token_program,
        &ctx.accounts.mint,
        &ctx.accounts.src_token,
        &ctx.accounts.custody_token,
        &ctx.accounts.transfer_authority,
        ctx.bumps["transfer_authority"],
        amount,
    )?;

    // Prepare Wormhole message. We need to normalize these amounts because we are working with
    // native assets.
    let mint = &ctx.accounts.mint;
    let token_address = mint.key().to_bytes();

    let decimals = Mint::parse(&mint.data.borrow()).unwrap().decimals();
    let token_transfer = crate::messages::TransferWithMessage {
        norm_amount: EncodedAmount::norm(U256::from(amount), decimals).0,
        token_address,
        token_chain: core_bridge_sdk::SOLANA_CHAIN,
        redeemer,
        redeemer_chain,
        sender,
        payload,
    };

    // Finally publish Wormhole message using the Core Bridge.
    post_token_bridge_message(
        ctx.accounts,
        ctx.bumps["core_emitter"],
        nonce,
        token_transfer,
    )
}
