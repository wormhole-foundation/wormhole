mod upgrade_contract;
pub use upgrade_contract::*;

use crate::error::TokenBridgeError;
use anchor_lang::prelude::*;
use core_bridge_program::{constants::SOLANA_CHAIN, zero_copy::PostedVaaV1};
use wormhole_raw_vaas::token_bridge::{TokenBridgeDecree, TokenBridgeGovPayload};

pub fn require_valid_posted_governance_vaa<'ctx>(
    vaa_key: &'ctx Pubkey,
    vaa_acc_data: &'ctx [u8],
) -> Result<TokenBridgeDecree<'ctx>> {
    crate::utils::require_valid_posted_vaa_key(vaa_key)?;

    let vaa = PostedVaaV1::parse(vaa_acc_data)?;

    require!(
        vaa.emitter_chain() == SOLANA_CHAIN
            && vaa.emitter_address() == crate::constants::GOVERNANCE_EMITTER,
        TokenBridgeError::InvalidGovernanceEmitter
    );

    TokenBridgeGovPayload::parse(vaa.payload())
        .map(|msg| msg.decree())
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))
}
