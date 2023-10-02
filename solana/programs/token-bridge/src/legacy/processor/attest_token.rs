use crate::{
    error::TokenBridgeError, legacy::instruction::LegacyAttestTokenArgs, utils, zero_copy::Mint,
};
use anchor_lang::prelude::*;
use anchor_spl::metadata;
use core_bridge_program::sdk::{self as core_bridge_sdk, LoadZeroCopy};

#[derive(Accounts)]
pub struct AttestToken<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// CHECK: Native mint. We ensure this mint is not one that has originated from a foreign
    /// network in access control.
    mint: AccountInfo<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _native_asset: UncheckedAccount<'info>,

    /// We derive this PDA because we do not involve the Token Metadata program with this
    /// instruction handler. It is the Token Bridge's job to verify that the metadata attested for
    /// is the correct one.
    #[account(
        seeds = [
            b"metadata",
            metadata::Metadata::id().as_ref(),
            mint.key().as_ref()
        ],
        bump,
        seeds::program = metadata::Metadata::id()
    )]
    token_metadata: Account<'info, metadata::MetadataAccount>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_message: Signer<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages.
    /// This PDA address is checked in `post_token_bridge_message`.
    core_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_fee_collector: Option<UncheckedAccount<'info>>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, core_bridge_sdk::cpi::CoreBridge>,
}

impl<'info> core_bridge_sdk::cpi::system_program::CreateAccount<'info> for AttestToken<'info> {
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::PublishMessage<'info> for AttestToken<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }

    fn core_bridge_config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn core_emitter_authority(&self) -> AccountInfo<'info> {
        self.core_emitter.to_account_info()
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

impl<'info>
    core_bridge_program::legacy::utils::ProcessLegacyInstruction<'info, LegacyAttestTokenArgs>
    for AttestToken<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyAttestToken";

    const ANCHOR_IX_FN: fn(Context<Self>, LegacyAttestTokenArgs) -> Result<()> = attest_token;

    fn order_account_infos<'a>(
        account_infos: &'a [AccountInfo<'info>],
    ) -> Result<Vec<AccountInfo<'info>>> {
        const NUM_ACCOUNTS: usize = 14;
        const CORE_BRIDGE_PROGRAM_IDX: usize = NUM_ACCOUNTS - 1;
        const SYSTEM_PROGRAM_IDX: usize = CORE_BRIDGE_PROGRAM_IDX - 1;

        let mut infos = account_infos.to_vec();

        // This check is inclusive because Core Bridge program, System program and Token program can
        // be in any order.
        if infos.len() >= NUM_ACCOUNTS {
            // System program needs to exist in these account infos.
            let system_program_idx = infos
                .iter()
                .position(|info| info.key() == anchor_lang::system_program::ID)
                .ok_or(error!(ErrorCode::InvalidProgramId))?;

            // Make sure System program is in the right index.
            if system_program_idx != SYSTEM_PROGRAM_IDX {
                infos.swap(SYSTEM_PROGRAM_IDX, system_program_idx);
            }

            // Core Bridge program needs to exist in these account infos.
            let core_bridge_program_idx = infos
                .iter()
                .position(|info| info.key() == core_bridge_program::ID)
                .ok_or(error!(ErrorCode::InvalidProgramId))?;

            // Make sure Token program is in the right index.
            if core_bridge_program_idx != CORE_BRIDGE_PROGRAM_IDX {
                infos.swap(CORE_BRIDGE_PROGRAM_IDX, core_bridge_program_idx);
            }
        }

        // Done.
        Ok(infos)
    }
}

impl<'info> AttestToken<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // Make sure the mint authority is not the Token Bridge's. If it is, then this mint
        // originated from a foreign network.
        let mint = Mint::load(&ctx.accounts.mint)?;
        require!(
            !crate::utils::is_wrapped_mint(&mint),
            TokenBridgeError::WrappedAsset
        );

        Ok(())
    }
}

#[access_control(AttestToken::constraints(&ctx))]
fn attest_token(ctx: Context<AttestToken>, args: LegacyAttestTokenArgs) -> Result<()> {
    let LegacyAttestTokenArgs { nonce } = args;

    let metadata = &ctx.accounts.token_metadata.data;
    let decimals = Mint::load(&ctx.accounts.mint).unwrap().decimals();

    // Finally post Wormhole message via Core Bridge.
    utils::cpi::post_token_bridge_message(
        ctx.accounts,
        &ctx.accounts.core_message,
        nonce,
        crate::messages::Attestation {
            token_address: ctx.accounts.mint.key().to_bytes(),
            token_chain: core_bridge_sdk::SOLANA_CHAIN,
            decimals,
            symbol: string_to_fixed32(&metadata.symbol),
            name: string_to_fixed32(&metadata.name),
        },
    )
}

pub(crate) fn string_to_fixed32(s: &String) -> [u8; 32] {
    let mut bytes = [0; 32];
    if s.len() > 32 {
        bytes.copy_from_slice(&s.as_bytes()[..32]);
    } else {
        bytes[..s.len()].copy_from_slice(s.as_bytes());
    }
    bytes
}
