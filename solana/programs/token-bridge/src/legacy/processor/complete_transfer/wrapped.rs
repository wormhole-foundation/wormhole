use crate::{
    constants::MINT_AUTHORITY_SEED_PREFIX,
    error::TokenBridgeError,
    legacy::instruction::EmptyArgs,
    state::{LegacyWrappedAsset, RegisteredEmitter},
    utils,
};
use anchor_lang::prelude::*;
use core_bridge_program::{
    legacy::utils::LegacyAnchorized,
    sdk::{self as core_bridge_sdk, LoadZeroCopy},
};
use wormhole_raw_vaas::token_bridge::TokenBridgeMessage;

#[derive(Accounts)]
pub struct CompleteTransferWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// CHECK: Posted VAA account, which will be read via zero-copy deserialization in the
    /// instruction handler, which also checks this account discriminator (so there is no need to
    /// check PDA seeds here).
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
    registered_emitter: Account<'info, LegacyAnchorized<0, RegisteredEmitter>>,

    /// CHECK: Recipient token account. Because we verify the wrapped mint, we can depend on the
    /// Token Program to mint the right tokens to this account because it requires that this mint
    /// equals the wrapped mint.
    #[account(mut)]
    recipient_token: AccountInfo<'info>,

    /// CHECK: Payer (relayer) token account. Because we verify the wrapped mint, we can depend on
    /// the Token Program to mint the right tokens to this account because it requires that this
    /// mint equals the wrapped mint.
    #[account(mut)]
    payer_token: AccountInfo<'info>,

    /// CHECK: Wrapped mint (i.e. minted by Token Bridge program).
    ///
    /// NOTE: Because this mint is guaranteed to have a Wrapped Asset account (since this account's
    /// pubkey is a part of the Wrapped Asset's PDA address), we do not need to check that this
    /// mint is one that the Token Bridge program has mint authority for.
    #[account(mut)]
    wrapped_mint: AccountInfo<'info>,

    /// Wrapped asset account, which is deserialized as its legacy representation. The latest
    /// version has an additional field (sequence number), which may not deserialize if wrapped
    /// metadata were not attested again to realloc this account. So we must deserialize this as the
    /// legacy representation.
    #[account(
        seeds = [LegacyWrappedAsset::SEED_PREFIX, wrapped_mint.key().as_ref()],
        bump,
    )]
    wrapped_asset: Account<'info, LegacyAnchorized<0, LegacyWrappedAsset>>,

    /// CHECK: This account is the authority that can burn and mint wrapped assets.
    #[account(
        seeds = [MINT_AUTHORITY_SEED_PREFIX],
        bump,
    )]
    mint_authority: AccountInfo<'info>,

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
    token_program: Program<'info, anchor_spl::token::Token>,
}

impl<'info> core_bridge_sdk::cpi::system_program::CreateAccount<'info>
    for CompleteTransferWrapped<'info>
{
    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }

    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }
}

impl<'info> utils::cpi::MintTo<'info> for CompleteTransferWrapped<'info> {
    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.wrapped_mint.to_account_info()
    }

    fn mint_authority(&self) -> AccountInfo<'info> {
        self.mint_authority.to_account_info()
    }
}

impl<'info> core_bridge_program::legacy::utils::ProcessLegacyInstruction<'info, EmptyArgs>
    for CompleteTransferWrapped<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyCompleteTransferWrapped";

    const ANCHOR_IX_FN: fn(Context<Self>, EmptyArgs) -> Result<()> = complete_transfer_wrapped;

    fn order_account_infos<'a>(
        account_infos: &'a [AccountInfo<'info>],
    ) -> Result<Vec<AccountInfo<'info>>> {
        super::order_complete_transfer_account_infos(account_infos)
    }
}

impl<'info> CompleteTransferWrapped<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let (token_chain, token_address) = super::validate_token_transfer_vaa(
            &ctx.accounts.vaa,
            &ctx.accounts.registered_emitter,
            &ctx.accounts.recipient_token,
            &ctx.accounts.recipient,
        )?;

        // For wrapped transfers, this token must have originated from another network.
        //
        // NOTE: This check may be redundant because our wrapped mint PDA should only exist for wrapped assets (i.e.
        // chain ID != 1. But there may be accounts that exist where the chain ID == 1, so we do perform this check as a
        // precaution).
        require_neq!(
            token_chain,
            core_bridge_sdk::SOLANA_CHAIN,
            TokenBridgeError::NativeAsset
        );

        // Wrapped asset account must agree with the encoded token info.
        let asset = &ctx.accounts.wrapped_asset;
        require!(
            token_chain == asset.token_chain && token_address == asset.token_address,
            TokenBridgeError::InvalidMint
        );

        // Done.
        Ok(())
    }
}

#[access_control(CompleteTransferWrapped::constraints(&ctx))]
fn complete_transfer_wrapped(
    ctx: Context<CompleteTransferWrapped>,
    _args: EmptyArgs,
) -> Result<()> {
    let vaa = core_bridge_sdk::VaaAccount::load(&ctx.accounts.vaa).unwrap();

    // Create the claim account to provide replay protection. Because this instruction creates this
    // account every time it is executed, this account cannot be created again with this emitter
    // address, chain and sequence combination.
    core_bridge_sdk::cpi::claim_vaa(ctx.accounts, &ctx.accounts.claim, &crate::ID, &vaa, None)?;

    let msg = TokenBridgeMessage::try_from(vaa.try_payload().unwrap()).unwrap();
    let transfer = msg.transfer().unwrap();

    // We do not have to denormalize wrapped mint amounts because by definition wrapped mints can
    // only have a max of 8 decimals, which is the same as the cap for normalized amounts.
    let mut mint_amount = transfer
        .encoded_amount()
        .0
        .try_into()
        .map_err(|_| TokenBridgeError::U64Overflow)?;
    let relayer_payout = transfer.encoded_relayer_fee().0.try_into().unwrap();

    // Save references to the token accounts to be used later.
    let recipient_token = &ctx.accounts.recipient_token;
    let payer_token = &ctx.accounts.payer_token;

    // Mint authority is who has the authority to mint.
    let mint_authority_seeds = &[MINT_AUTHORITY_SEED_PREFIX, &[ctx.bumps["mint_authority"]]];

    // If there is a payout to the relayer and the relayer's token account differs from the transfer
    // recipient's, we have to make an extra mint.
    if relayer_payout > 0 && recipient_token.key() != payer_token.key() {
        // NOTE: This math operation is safe because the relayer payout is always <= to the
        // total outbound transfer amount.
        mint_amount -= relayer_payout;

        utils::cpi::mint_to(
            ctx.accounts,
            payer_token,
            relayer_payout,
            Some(&[mint_authority_seeds]),
        )?;
    }

    // If there is any amount left after the relayer payout, finally mint remaining.
    utils::cpi::mint_to(
        ctx.accounts,
        recipient_token,
        mint_amount,
        Some(&[mint_authority_seeds]),
    )
}
