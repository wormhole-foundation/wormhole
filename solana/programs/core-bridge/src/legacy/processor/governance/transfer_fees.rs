use crate::{
    constants::{FEE_COLLECTOR_SEED_PREFIX, SOLANA_CHAIN},
    error::CoreBridgeError,
    legacy::{instruction::EmptyArgs, utils::LegacyAnchorized},
    state::Config,
    zero_copy::{LoadZeroCopy, VaaAccount},
};
use anchor_lang::{
    prelude::*,
    system_program::{self, Transfer},
};
use ruint::aliases::U256;
use wormhole_raw_vaas::core::CoreBridgeGovPayload;

#[derive(Accounts)]
pub struct TransferFees<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// For governance VAAs, we need to make sure that the current guardian set was used to attest
    /// for this governance decree.
    #[account(
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, LegacyAnchorized<0, Config>>,

    /// CHECK: Posted VAA account, which will be read via zero-copy deserialization in the
    /// instruction handler, which also checks this account discriminator (so there is no need to
    /// check PDA seeds here).
    vaa: AccountInfo<'info>,

    /// CHECK: Account representing that a VAA has been consumed. Seeds are checked when
    /// [claim_vaa](crate::utils::vaa::claim_vaa) is called.
    #[account(mut)]
    claim: AccountInfo<'info>,

    /// CHECK: Fee collector. Fees will be collected by transferring lamports from this account to
    /// the recipient.
    #[account(
        mut,
        seeds = [FEE_COLLECTOR_SEED_PREFIX],
        bump,
    )]
    fee_collector: AccountInfo<'info>,

    /// CHECK: This recipient account must equal the one encoded in the governance VAA.
    #[account(mut)]
    recipient: AccountInfo<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

impl<'info> crate::utils::cpi::CreateAccount<'info> for TransferFees<'info> {
    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }

    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }
}

impl<'info> crate::legacy::utils::ProcessLegacyInstruction<'info, EmptyArgs>
    for TransferFees<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyTransferFees";

    const ANCHOR_IX_FN: fn(Context<Self>, EmptyArgs) -> Result<()> = transfer_fees;
}

impl<'info> TransferFees<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let vaa = VaaAccount::load(&ctx.accounts.vaa)?;
        let gov_payload = super::require_valid_governance_vaa(&ctx.accounts.config, &vaa)?;

        let decree = gov_payload
            .transfer_fees()
            .ok_or(error!(CoreBridgeError::InvalidGovernanceAction))?;

        // Make sure that transferring fees is intended for this network.
        require_eq!(
            decree.chain(),
            SOLANA_CHAIN,
            CoreBridgeError::GovernanceForAnotherChain
        );

        // Make sure that the encoded fee does not overflow since the encoded amount is u256 (and
        // lamports are u64).
        let amount = U256::from_be_bytes(decree.amount());
        require!(amount <= U256::from(u64::MAX), CoreBridgeError::U64Overflow);

        // The recipient provided in the account context must be the same as the one encoded in the
        // governance VAA.
        require_keys_eq!(
            ctx.accounts.recipient.key(),
            Pubkey::from(decree.recipient()),
            CoreBridgeError::InvalidFeeRecipient
        );

        // We cannot remove more than what is required to be rent exempt. We prefer to abort here
        // with an explicit Core Bridge error rather than abort when we attempt the transfer (since
        // the transfer will fail if the lamports in the fee collector account drops below being
        // rent exempt).
        {
            let (data_len, lamports) = {
                let fee_collector = AsRef::<AccountInfo>::as_ref(&ctx.accounts.fee_collector);
                (fee_collector.data_len(), fee_collector.lamports())
            };
            let min_required = Rent::get().map(|rent| rent.minimum_balance(data_len))?;
            require!(
                lamports.saturating_sub(to_u64_unchecked(&amount)) >= min_required,
                CoreBridgeError::NotEnoughLamports
            );
        }

        // Done.
        Ok(())
    }
}

#[access_control(TransferFees::constraints(&ctx))]
fn transfer_fees(ctx: Context<TransferFees>, _args: EmptyArgs) -> Result<()> {
    let vaa = VaaAccount::load(&ctx.accounts.vaa).unwrap();

    // Create the claim account to provide replay protection. Because this instruction creates this
    // account every time it is executed, this account cannot be created again with this emitter
    // address, chain and sequence combination.
    crate::utils::vaa::claim_vaa(ctx.accounts, &ctx.accounts.claim, &crate::ID, &vaa)?;

    let gov_payload = CoreBridgeGovPayload::try_from(vaa.try_payload().unwrap())
        .unwrap()
        .decree();
    let decree = gov_payload.transfer_fees().unwrap();

    let fee_collector = AsRef::<AccountInfo>::as_ref(&ctx.accounts.fee_collector);

    // Finally transfer collected fees to recipient.
    system_program::transfer(
        CpiContext::new_with_signer(
            ctx.accounts.system_program.to_account_info(),
            Transfer {
                from: fee_collector.to_account_info(),
                to: ctx.accounts.recipient.to_account_info(),
            },
            &[&[FEE_COLLECTOR_SEED_PREFIX, &[ctx.bumps["fee_collector"]]]],
        ),
        to_u64_unchecked(&U256::from_be_bytes(decree.amount())),
    )?;

    // Done.
    Ok(())
}

/// Uint encodes limbs in little endian, so we will take the first u64 value.
fn to_u64_unchecked(value: &U256) -> u64 {
    value.as_limbs()[0]
}
