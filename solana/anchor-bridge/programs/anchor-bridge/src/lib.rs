use anchor_lang::{prelude::*, solana_program};

mod account;
mod api;

use account::BridgeInfo;
use account::GuardianSetInfo;

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
pub struct VerifySig<'info> {
    pub bridge: AccountInfo<'info>,
    pub system: AccountInfo<'info>,
    pub instruction_sysvar: AccountInfo<'info>,
    pub bridge_info: ProgramState<'info, BridgeInfo>,
    pub sig_info: AccountInfo<'info>,
    pub guardian_set_info: ProgramState<'info, GuardianSetInfo>,
    pub payer_info: AccountInfo<'info>,
}

#[program]
pub mod anchor_bridge {
    use super::*;

    pub fn verify_signatures(ctx: Context<VerifySig>, data: VerifySigsData) -> ProgramResult {
        api::verify_signatures(ctx, data)
    }
}
