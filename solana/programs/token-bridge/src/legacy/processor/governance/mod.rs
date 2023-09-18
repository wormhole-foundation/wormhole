mod upgrade_contract;
pub use upgrade_contract::*;

use crate::error::TokenBridgeError;
use anchor_lang::prelude::*;
use core_bridge_program::sdk::{self as core_bridge_sdk};
use wormhole_raw_vaas::token_bridge::{TokenBridgeDecree, TokenBridgeGovPayload};

pub fn require_valid_posted_governance_vaa<'ctx>(
    vaa_key: &'ctx Pubkey,
    vaa: &'ctx core_bridge_sdk::VaaAccount<'ctx>,
) -> Result<TokenBridgeDecree<'ctx>> {
    crate::utils::require_valid_posted_vaa_key(vaa_key)?;

    let (emitter_address, emitter_chain, _) = vaa.try_emitter_info()?;
    require!(
        emitter_chain == core_bridge_sdk::SOLANA_CHAIN
            && emitter_address == crate::constants::GOVERNANCE_EMITTER,
        TokenBridgeError::InvalidGovernanceEmitter
    );

    TokenBridgeGovPayload::try_from(vaa.try_payload().unwrap())
        .map(|msg| msg.decree())
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))
}
