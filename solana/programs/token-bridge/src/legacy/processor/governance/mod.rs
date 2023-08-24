mod register_chain;
pub use register_chain::*;

mod upgrade_contract;
pub use upgrade_contract::*;

use crate::error::TokenBridgeError;
use anchor_lang::prelude::*;
use core_bridge_program::{
    constants::SOLANA_CHAIN,
    state::{PartialPostedVaaV1, VaaAccountDetails},
};
use wormhole_raw_vaas::token_bridge::TokenBridgeGovPayload;

pub fn require_valid_governance_posted_vaa(
    vaa_details: VaaAccountDetails,
    vaa_acc_data: &[u8],
) -> Result<TokenBridgeGovPayload<'_>> {
    let VaaAccountDetails {
        key,
        emitter_chain,
        emitter_address,
        sequence: _,
    } = vaa_details;
    crate::utils::require_valid_posted_vaa_key(&key)?;

    require!(
        emitter_chain == SOLANA_CHAIN && emitter_address == crate::constants::GOVERNANCE_EMITTER,
        TokenBridgeError::InvalidGovernanceEmitter
    );

    parse_acc_data(vaa_acc_data)
}

pub(super) fn parse_acc_data(vaa_acc_data: &[u8]) -> Result<TokenBridgeGovPayload<'_>> {
    TokenBridgeGovPayload::parse(&vaa_acc_data[PartialPostedVaaV1::PAYLOAD_START..])
        .map_err(|_| TokenBridgeError::InvalidGovernanceVaa.into())
}
