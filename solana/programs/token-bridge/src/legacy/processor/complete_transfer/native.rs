use crate::{
    constants::CUSTODY_AUTHORITY_SEED_PREFIX, error::TokenBridgeError,
    legacy::instruction::EmptyArgs, state::RegisteredEmitter,
};
use anchor_lang::prelude::*;
use anchor_spl::token;
use core_bridge_program::sdk as core_bridge;
use wormhole_raw_vaas::token_bridge::TokenBridgeMessage;

#[derive(Accounts)]
pub struct CompleteTransferNative<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// CHECK: Posted VAA account, which will be read via zero-copy deserialization in the
    /// instruction handler, which also checks this account discriminator (so there is no need to
    /// check PDA seeds here).
    #[account(owner = core_bridge::id())]
    vaa: AccountInfo<'info>,

    /// CHECK: Account representing that a VAA has been consumed. Seeds are checked when
    /// [claim_vaa](core_bridge_sdk::cpi::claim_vaa) is called.
    #[account(mut)]
    claim: AccountInfo<'info>,

    /// This account is a foreign token Bridge and is created via the Register Chain governance
    /// decree.
    ///
    /// NOTE: The seeds of this account are insane because they include the emitter address, which
    /// allows registering multiple emitter addresses for the same chain ID. These seeds are not
    /// checked via Anchor macro, but will be checked in the access control function instead.
    ///
    /// See the `require_valid_token_bridge_vaa` instruction handler for more details.
    registered_emitter: Account<'info, core_bridge::legacy::LegacyAnchorized<RegisteredEmitter>>,

    /// Recipient token account.
    #[account(
        mut,
        token::mint = mint
    )]
    recipient_token: Account<'info, token::TokenAccount>,

    /// CHECK: Payer (relayer) token account. Because we check the mint of the custody token
    /// account, we can be sure that this token account is the same mint since the Token Program
    /// transfer instruction handler checks that the mints of these two accounts must be the same.
    ///
    /// NOTE: We will check that the owner of this account belongs to the payer of this transaction.
    #[account(
        mut,
        token::mint = mint,
        token::authority = payer
    )]
    payer_token: Account<'info, token::TokenAccount>,

    /// Custody token account.
    ///
    /// CHECK: Because we are deriving this PDA's address, we ensure that this account is the Token
    /// Bridge's custody token account. And because this account can only be created on a native
    /// mint's outbound transfer (since these tokens originated from Solana), this account should
    /// already be created.
    #[account(
        mut,
        seeds = [mint.key().as_ref()],
        bump,
    )]
    custody_token: AccountInfo<'info>,

    /// Native mint. We know this mint does not belong to the Token Bridge because a custody token
    /// account exists for it.
    mint: Account<'info, token::Mint>,

    /// CHECK: This account is the authority that can move tokens from the custody account.
    #[account(
        seeds = [CUSTODY_AUTHORITY_SEED_PREFIX],
        bump,
    )]
    custody_authority: AccountInfo<'info>,

    /// CHECK: Expected recipient, which is the owner of the recipient token account. This account
    /// does not need to be provided if the recipient encoded in the VAA is the token account
    /// provided above.
    ///
    /// NOTE: In the old implementation, this account used to be the rent sysvar. Because this
    /// sysvar is no longer needed for any instruction handler, we are repurposing this account. So
    /// for integrators that have been passing the rent pubkey here, it is expected that the token
    /// transfer VAA they redeem has the token account encoded in its VAA. Otherwise, they will
    /// break.
    recipient: Option<AccountInfo<'info>>,

    system_program: Program<'info, System>,
    token_program: Program<'info, token::Token>,
}

impl<'info> core_bridge::legacy::ProcessLegacyInstruction<'info, EmptyArgs>
    for CompleteTransferNative<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyCompleteTransferNative";

    const ANCHOR_IX_FN: fn(Context<Self>, EmptyArgs) -> Result<()> = complete_transfer_native;

    fn order_account_infos<'a>(
        account_infos: &'a [AccountInfo<'info>],
    ) -> Result<Vec<AccountInfo<'info>>> {
        super::order_complete_transfer_account_infos(account_infos)
    }
}

impl<'info> CompleteTransferNative<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let (token_chain, token_address) = super::validate_token_transfer_vaa(
            &ctx.accounts.vaa,
            &ctx.accounts.registered_emitter,
            &ctx.accounts.recipient_token,
            &ctx.accounts.recipient,
        )?;

        // For native transfers, this mint must have been created on Solana.
        require_eq!(
            token_chain,
            core_bridge::SOLANA_CHAIN,
            TokenBridgeError::WrappedAsset
        );

        // Mint account must agree with the encoded token address.
        require_keys_eq!(
            ctx.accounts.mint.key(),
            Pubkey::from(token_address),
            TokenBridgeError::InvalidMint
        );

        // Done.
        Ok(())
    }
}

#[access_control(CompleteTransferNative::constraints(&ctx))]
fn complete_transfer_native(ctx: Context<CompleteTransferNative>, _args: EmptyArgs) -> Result<()> {
    let vaa = core_bridge::VaaAccount::load(&ctx.accounts.vaa).unwrap();

    // Create the claim account to provide replay protection. Because this instruction creates this
    // account every time it is executed, this account cannot be created again with this emitter
    // address, chain and sequence combination.
    core_bridge::claim_vaa(
        CpiContext::new(
            ctx.accounts.system_program.to_account_info(),
            core_bridge::ClaimVaa {
                claim: ctx.accounts.claim.to_account_info(),
                payer: ctx.accounts.payer.to_account_info(),
            },
        ),
        &crate::ID,
        &vaa,
        None,
    )?;

    let msg = TokenBridgeMessage::try_from(vaa.try_payload().unwrap()).unwrap();
    let transfer = msg.transfer().unwrap();

    let decimals = ctx.accounts.mint.decimals;

    // Denormalize transfer transfer_amount and relayer payouts based on this mint's decimals. When these
    // transfers were made outbound, the amounts were normalized, so it is safe to unwrap these
    // operations.
    let mut transfer_amount = transfer
        .encoded_amount()
        .denorm(decimals)
        .try_into()
        .expect("Solana token amounts are u64");
    let relayer_payout = transfer
        .encoded_relayer_fee()
        .denorm(decimals)
        .try_into()
        .unwrap();

    // Save references to these accounts to be used later.
    let recipient_token = &ctx.accounts.recipient_token;
    let payer_token = &ctx.accounts.payer_token;
    let custody_token = &ctx.accounts.custody_token;

    // Custody authority is who has the authority to transfer tokens from the custody account.
    let custody_authority_seeds = &[
        CUSTODY_AUTHORITY_SEED_PREFIX,
        &[ctx.bumps["custody_authority"]],
    ];

    // If there is a payout to the relayer and the relayer's token account differs from the transfer
    // recipient's, we have to make an extra transfer.
    if relayer_payout > 0 && recipient_token.key() != payer_token.key() {
        // NOTE: This math operation is safe because the relayer payout is always <= to the
        // total outbound transfer transfer_amount.
        transfer_amount -= relayer_payout;

        token::transfer(
            CpiContext::new_with_signer(
                ctx.accounts.token_program.to_account_info(),
                token::Transfer {
                    from: custody_token.to_account_info(),
                    to: payer_token.to_account_info(),
                    authority: ctx.accounts.custody_authority.to_account_info(),
                },
                &[custody_authority_seeds],
            ),
            relayer_payout,
        )?;
    }

    // Finally transfer remaining transfer_amount to recipient.
    token::transfer(
        CpiContext::new_with_signer(
            ctx.accounts.token_program.to_account_info(),
            token::Transfer {
                from: custody_token.to_account_info(),
                to: recipient_token.to_account_info(),
                authority: ctx.accounts.custody_authority.to_account_info(),
            },
            &[custody_authority_seeds],
        ),
        transfer_amount,
    )
}
