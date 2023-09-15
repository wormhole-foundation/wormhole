use crate::{
    constants::CUSTODY_AUTHORITY_SEED_PREFIX,
    error::TokenBridgeError,
    legacy::instruction::EmptyArgs,
    state::{Claim, RegisteredEmitter},
    utils,
    zero_copy::Mint,
};
use anchor_lang::prelude::*;
use core_bridge_program::{
    constants::SOLANA_CHAIN, legacy::utils::LegacyAnchorized, zero_copy::PostedVaaV1,
};
use wormhole_raw_vaas::token_bridge::TokenBridgeMessage;

#[derive(Accounts)]
pub struct CompleteTransferWithPayloadNative<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// CHECK: Posted VAA account, which will be read via zero-copy deserialization in the
    /// instruction handler, which also checks this account discriminator (so there is no need to
    /// check PDA seeds here).
    #[account(owner = core_bridge_program::ID)]
    posted_vaa: AccountInfo<'info>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            PostedVaaV1::parse(&posted_vaa)
                .map(|vaa| vaa.emitter_address())?
                .as_ref(),
            PostedVaaV1::parse(&posted_vaa)
                .map(|vaa| vaa.emitter_chain().to_be_bytes())?
                .as_ref(),
            PostedVaaV1::parse(&posted_vaa)
                .map(|vaa| vaa.sequence().to_be_bytes())?
                .as_ref(),
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
    registered_emitter: Account<'info, LegacyAnchorized<0, RegisteredEmitter>>,

    /// CHECK: Destination token account. Because we check the mint of the custody token account, we
    /// can be sure that this token account is the same mint since the Token Program transfer
    /// instruction handler checks that the mints of these two accounts must be the same.
    #[account(mut)]
    dst_token: AccountInfo<'info>,

    redeemer_authority: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _relayer_fee_token: UncheckedAccount<'info>,

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

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_program: Program<'info, anchor_spl::token::Token>,
}

impl<'info> utils::cpi::Transfer<'info> for CompleteTransferWithPayloadNative<'info> {
    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn from(&self) -> Option<AccountInfo<'info>> {
        Some(self.custody_token.to_account_info())
    }

    fn authority(&self) -> Option<AccountInfo<'info>> {
        Some(self.custody_authority.to_account_info())
    }
}

impl<'info> core_bridge_program::legacy::utils::ProcessLegacyInstruction<'info, EmptyArgs>
    for CompleteTransferWithPayloadNative<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyCompleteTransferWithPayloadNative";

    const ANCHOR_IX_FN: fn(Context<Self>, EmptyArgs) -> Result<()> =
        complete_transfer_with_payload_native;

    fn order_account_infos<'a>(
        account_infos: &'a [AccountInfo<'info>],
    ) -> Result<Vec<AccountInfo<'info>>> {
        super::order_complete_transfer_with_payload_account_infos(account_infos)
    }
}

impl<'info> CompleteTransferWithPayloadNative<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // Make sure the mint authority is not the Token Bridge's. If it is, then this mint
        // originated from a foreign network.
        crate::utils::require_native_mint(&ctx.accounts.mint)?;

        let (token_chain, token_address) = super::validate_posted_token_transfer_with_payload(
            &ctx.accounts.posted_vaa,
            &ctx.accounts.registered_emitter,
            &ctx.accounts.redeemer_authority,
            &ctx.accounts.dst_token,
        )?;

        // For native transfers, this mint must have been created on Solana.
        require_eq!(token_chain, SOLANA_CHAIN, TokenBridgeError::WrappedAsset);

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

#[access_control(CompleteTransferWithPayloadNative::constraints(&ctx))]
fn complete_transfer_with_payload_native(
    ctx: Context<CompleteTransferWithPayloadNative>,
    _args: EmptyArgs,
) -> Result<()> {
    // Mark the claim as complete. The account only exists to ensure that the VAA is not processed,
    // so this value does not matter. But the legacy program set this data to true.
    ctx.accounts.claim.is_complete = true;

    let vaa = PostedVaaV1::parse_unchecked(&ctx.accounts.posted_vaa);

    // Denormalize transfer amount based on this mint's decimals. When these transfers were made
    // outbound, the amounts were normalized, so it is safe to unwrap these operations.
    let transfer_amount = TokenBridgeMessage::parse(vaa.payload())
        .unwrap()
        .transfer_with_message()
        .unwrap()
        .encoded_amount()
        .denorm(Mint::parse_unchecked(&ctx.accounts.mint).decimals())
        .try_into()
        .expect("Solana token amounts are u64");

    // Finally transfer encoded amount.
    utils::cpi::transfer(
        ctx.accounts,
        ctx.accounts.dst_token.to_account_info(),
        transfer_amount,
        Some(&[&[
            CUSTODY_AUTHORITY_SEED_PREFIX,
            &[ctx.bumps["custody_authority"]],
        ]]),
    )
}
