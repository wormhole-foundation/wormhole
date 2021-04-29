use anchor_lang::{prelude::*, solana_program};

mod account;
mod api;
mod types;

use account::{BridgeInfo, GuardianSetInfo};
use types::BridgeConfig;

/// An enum with labeled network identifiers. These must be consistent accross all wormhole
/// contracts deployed on each chain.
#[repr(u8)]
pub enum Chain {
    Solana = 1u8,
}

/// chain id of this chain
pub const CHAIN_ID_SOLANA: u8 = Chain::Solana as u8;
/// maximum number of guardians
pub const MAX_LEN_GUARDIAN_KEYS: usize = 20;

#[derive(Accounts)]
pub struct VerifySig<'info> {
    pub bridge: AccountInfo<'info>,
    pub system: AccountInfo<'info>,
    pub instruction_sysvar: AccountInfo<'info>,
    pub bridge_info: ProgramState<'info, BridgeInfo>,
    pub sig_info: AccountInfo<'info>,
    pub guardian_set_info: ProgramState<'info, GuardianSetInfo>,
    pub payer_info: AccountInfo<'info>,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, Debug)]
pub struct VerifySigsData {
    /// hash of the VAA
    pub hash: [u8; 32],
    /// instruction indices of signers (-1 for missing)
    pub signers: [i8; MAX_LEN_GUARDIAN_KEYS],
    /// indicates whether this verification should only succeed if the sig account does not exist
    pub initial_creation: bool,
}

#[derive(Accounts)]
pub struct Initialize {}

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, Debug)]
pub struct InitializeData {
    /// number of initial guardians
    pub len_guardians: u8,
    /// guardians that are allowed to sign mints
    pub initial_guardian_keys: [[u8; 20]; MAX_LEN_GUARDIAN_KEYS],
    /// config for the bridge
    pub config: BridgeConfig,
}

#[program]
pub mod anchor_bridge {
    use super::*;

    pub fn initialize(ctx: Context<Initialize>, data: InitializeData) -> ProgramResult {
        api::initialize(
            ctx,
            data.len_guardians,
            data.initial_guardian_keys,
            data.config,
        )
    }

    pub fn verify_signatures(ctx: Context<VerifySig>, data: VerifySigsData) -> ProgramResult {
        api::verify_signatures(ctx, data)
    }
}
