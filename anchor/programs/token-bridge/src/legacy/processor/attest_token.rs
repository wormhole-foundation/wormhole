use crate::{
    constants::EMITTER_SEED_PREFIX,
    legacy::LegacyAttestTokenArgs,
    processor::{post_token_bridge_message, PostTokenBridgeMessage},
    utils,
};
use anchor_lang::prelude::*;
use anchor_spl::{
    metadata::{Metadata, MetadataAccount},
    token::Mint,
};
use core_bridge_program::{self, constants::SOLANA_CHAIN, state::BridgeProgramData, CoreBridge};
use wormhole_io::Writeable;
use wormhole_solana_common::SeedPrefix;

use std::io;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Attestation {
    pub token_address: [u8; 32],
    pub token_chain: u16,
    pub decimals: u8,

    pub symbol: [u8; 32],
    pub name: [u8; 32],
}

impl Writeable for Attestation {
    fn written_size(&self) -> usize {
        1 + 32 + 2 + 1 + 32 + 32
    }

    fn write<W>(&self, writer: &mut W) -> io::Result<()>
    where
        Self: Sized,
        W: io::Write,
    {
        self.token_address.write(writer)?;
        self.token_chain.write(writer)?;
        self.decimals.write(writer)?;
        self.symbol.write(writer)?;
        self.name.write(writer)?;
        Ok(())
    }
}

#[derive(Accounts)]
pub struct AttestToken<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    mint: Account<'info, Mint>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _native_asset: UncheckedAccount<'info>,

    #[account(
        seeds = [
            b"metadata",
            Metadata::id().as_ref(),
            mint.key().as_ref()
        ],
        bump,
        seeds::program = Metadata::id()
    )]
    token_metadata: Account<'info, MetadataAccount>,

    /// We need to deserialize this account to determine the Wormhole message fee.
    #[account(
        seeds = [BridgeProgramData::seed_prefix()],
        bump,
        seeds::program = core_bridge_program
    )]
    core_bridge: Account<'info, BridgeProgramData>,

    /// CHECK: This account is needed for the Core Bridge program.
    core_message: UncheckedAccount<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages.
    #[account(
        seeds = [EMITTER_SEED_PREFIX],
        bump,
    )]
    core_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    core_fee_collector: UncheckedAccount<'info>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, CoreBridge>,
}

impl<'info> AttestToken<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        // Make sure the mint authority is not the Token Bridge's. If it is, then this mint
        // originated from a foreign network.
        utils::require_native_mint(&ctx.accounts.mint)
    }
}

#[access_control(AttestToken::accounts(&ctx))]
pub fn attest_token(ctx: Context<AttestToken>, args: LegacyAttestTokenArgs) -> Result<()> {
    let LegacyAttestTokenArgs { nonce } = args;

    let metadata = &ctx.accounts.token_metadata.data;

    // Finally post Wormhole message via Core Bridge.
    post_token_bridge_message(
        PostTokenBridgeMessage {
            core_bridge: &ctx.accounts.core_bridge,
            core_message: &ctx.accounts.core_message,
            core_emitter: &ctx.accounts.core_emitter,
            core_emitter_sequence: &ctx.accounts.core_emitter_sequence,
            payer: &ctx.accounts.payer,
            core_fee_collector: &ctx.accounts.core_fee_collector,
            system_program: &ctx.accounts.system_program,
            core_bridge_program: &ctx.accounts.core_bridge_program,
        },
        ctx.bumps["core_emitter"],
        nonce,
        Attestation {
            token_address: ctx.accounts.mint.key().to_bytes().into(),
            token_chain: SOLANA_CHAIN,
            decimals: ctx.accounts.mint.decimals,
            symbol: string_to_fixed32(&metadata.symbol),
            name: string_to_fixed32(&metadata.name),
        },
    )
}

fn string_to_fixed32(s: &String) -> [u8; 32] {
    let mut bytes = [0; 32];
    if s.len() > 32 {
        bytes.copy_from_slice(&s.as_bytes()[..32]);
    } else {
        bytes[..s.len()].copy_from_slice(s.as_bytes());
    }
    bytes
}
