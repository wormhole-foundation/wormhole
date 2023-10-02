mod register_chain;
pub use register_chain::*;

mod secure_registered_emitter;
pub use secure_registered_emitter::*;

use crate::error::TokenBridgeError;
use anchor_lang::prelude::*;
use core_bridge_program::sdk::{self as core_bridge_sdk};
use wormhole_raw_vaas::token_bridge::{TokenBridgeDecree, TokenBridgeGovPayload};

pub fn require_valid_governance_vaa<'ctx>(
    vaa_key: &'ctx Pubkey,
    vaa: &'ctx core_bridge_sdk::VaaAccount<'ctx>,
) -> Result<TokenBridgeDecree<'ctx>> {
    crate::utils::require_valid_vaa_key(vaa_key)?;

    let (emitter_address, emitter_chain, _) = vaa.try_emitter_info()?;
    require!(
        emitter_chain == crate::constants::GOVERNANCE_CHAIN
            && emitter_address == crate::constants::GOVERNANCE_EMITTER,
        TokenBridgeError::InvalidGovernanceEmitter
    );

    // Because emitter_chain and emitter_address getters have succeeded, we can safely unwrap this
    // payload call.
    TokenBridgeGovPayload::try_from(vaa.try_payload().unwrap())
        .map(|msg| msg.decree())
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))
}
