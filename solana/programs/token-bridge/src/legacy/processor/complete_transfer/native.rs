use crate::{
    constants::CUSTODY_AUTHORITY_SEED_PREFIX,
    error::TokenBridgeError,
    legacy::EmptyArgs,
    processor::withdraw_native_tokens,
    state::{Claim, RegisteredEmitter},
    zero_copy::Mint,
};
use anchor_lang::prelude::*;
use anchor_spl::token;
use core_bridge_program::{
    constants::SOLANA_CHAIN, legacy::utils::LegacyAnchorized, sdk::cpi::CoreBridge,
    zero_copy::PostedVaaV1,
};
use wormhole_raw_vaas::token_bridge::TokenBridgeMessage;

#[derive(Accounts)]
pub struct CompleteTransferNative<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// CHECK: We will be performing zero-copy deserialization in the instruction handler.
    #[account(
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.message_hash().as_ref()
        ],
        bump,
        seeds::program = core_bridge_program
    )]
    posted_vaa: AccountInfo<'info>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.emitter_address().as_ref(),
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.emitter_chain().to_be_bytes().as_ref(),
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.sequence().to_be_bytes().as_ref(),
        ],
        bump,
    )]
    claim: Account<'info, LegacyAnchorized<0, Claim>>,

    /// This account is a foreign token Bridge and is created via the Register Chain governance
    /// decree.
    ///
    /// NOTE: The seeds of this account are insane because they include the emitter address, which
    /// allows registering multiple emitter addresses for the same chain ID. These seeds are not
    /// checked via Anchor macro, but will be checked in the access control function instead.
    ///
    /// See the `require_valid_token_bridge_posted_vaa` instruction handler for more details.
    registered_emitter: Box<Account<'info, LegacyAnchorized<0, RegisteredEmitter>>>,

    /// CHECK: Recipient token account. Because we check the mint of the custody token account, we
    /// can be sure that this token account is the same mint since the Token Program transfer
    /// instruction handler checks that the mints of these two accounts must be the same.
    #[account(mut)]
    recipient_token: AccountInfo<'info>,

    /// CHECK: Payer (relayer) token account. Because we check the mint of the custody token
    /// account, we can be sure that this token account is the same mint since the Token Program
    /// transfer instruction handler checks that the mints of these two accounts must be the same.
    ///
    /// NOTE: We will check that the owner of this account belongs to the payer of this transaction.
    #[account(mut)]
    payer_token: AccountInfo<'info>,

    /// CHECK: Custody token account. Because we are deriving this PDA's address, we ensure that
    /// this account is the Token Bridge's custody token account. And because this account can only
    /// be created on a native mint's outbound transfer (since these tokens originated from Solana),
    /// this account should already be created.
    #[account(
        mut,
        seeds = [mint.key().as_ref()],
        bump,
    )]
    custody_token: AccountInfo<'info>,

    /// CHECK: Native mint. We ensure this mint is not one that has originated from a foreign
    /// network in access control.
    mint: AccountInfo<'info>,

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
    core_bridge_program: Program<'info, CoreBridge>,
    token_program: Program<'info, token::Token>,
}

impl<'info> core_bridge_program::legacy::utils::ProcessLegacyInstruction<'info, EmptyArgs>
    for CompleteTransferNative<'info>
{
    const LOG_IX_NAME: &'static str = "LegacCompleteTransferNative";

    const ANCHOR_IX_FN: fn(Context<Self>, EmptyArgs) -> Result<()> = complete_transfer_native;
}

impl<'info> CompleteTransferNative<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        require_keys_eq!(
            crate::zero_copy::TokenAccount::parse(&ctx.accounts.payer_token.try_borrow_data()?)?
                .owner(),
            ctx.accounts.payer.key(),
            ErrorCode::ConstraintTokenOwner
        );

        // Make sure the mint authority is not the Token Bridge's. If it is, then this mint
        // originated from a foreign network.
        crate::utils::require_native_mint(&ctx.accounts.mint)?;

        let vaa = &ctx.accounts.posted_vaa;
        let vaa_key = vaa.key();
        let acc_data = vaa.try_borrow_data()?;
        let transfer = super::validate_posted_token_transfer(
            &vaa_key,
            &acc_data,
            &ctx.accounts.registered_emitter,
            &ctx.accounts.recipient_token,
            &ctx.accounts.recipient,
        )?;

        // For native transfers, this mint must have been created on Solana.
        require_eq!(
            transfer.token_chain(),
            SOLANA_CHAIN,
            TokenBridgeError::WrappedAsset
        );

        // Mint account must agree with the encoded token address.
        require_eq!(
            ctx.accounts.mint.key(),
            Pubkey::from(transfer.token_address()),
            TokenBridgeError::InvalidMint
        );

        // Done.
        Ok(())
    }
}

#[access_control(CompleteTransferNative::constraints(&ctx))]
fn complete_transfer_native(ctx: Context<CompleteTransferNative>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete. The account only exists to ensure that the VAA is not processed,
    // so this value does not matter. But the legacy program set this data to true.
    ctx.accounts.claim.is_complete = true;

    let acc_data = ctx.accounts.posted_vaa.data.borrow();
    let vaa = PostedVaaV1::parse(&acc_data).unwrap();
    let msg = TokenBridgeMessage::parse(vaa.payload()).unwrap();
    let transfer = msg.transfer().unwrap();

    let decimals = Mint::parse(&ctx.accounts.mint.data.borrow())
        .unwrap()
        .decimals();

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
    let token_program = &ctx.accounts.token_program;
    let custody_token = &ctx.accounts.custody_token;
    let custody_authority = &ctx.accounts.custody_authority;
    let recipient_token = &ctx.accounts.recipient_token;
    let payer_token = &ctx.accounts.payer_token;

    // Custody authority is who has the authority to transfer tokens from the custody account.
    let custody_authority_bump = ctx.bumps["custody_authority"];

    // If there is a payout to the relayer and the relayer's token account differs from the transfer
    // recipient's, we have to make an extra transfer.
    if relayer_payout > 0 && recipient_token.key() != payer_token.key() {
        // NOTE: This math operation is safe because the relayer payout is always <= to the
        // total outbound transfer transfer_amount.
        transfer_amount -= relayer_payout;

        withdraw_native_tokens(
            token_program,
            custody_token,
            payer_token,
            custody_authority,
            custody_authority_bump,
            relayer_payout,
        )?;
    }

    // Finally transfer remaining transfer_amount to recipient.
    withdraw_native_tokens(
        token_program,
        custody_token,
        recipient_token,
        custody_authority,
        custody_authority_bump,
        transfer_amount,
    )
}
