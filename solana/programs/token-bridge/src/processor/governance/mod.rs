mod register_chain;
pub use register_chain::*;

use crate::error::TokenBridgeError;
use anchor_lang::prelude::*;
use core_bridge_program::{constants::SOLANA_CHAIN, state::ZeroCopyEncodedVaa};
use wormhole_raw_vaas::{token_bridge::TokenBridgeGovPayload, Payload};

pub fn require_valid_governance_encoded_vaa(
    vaa_acc_data: &[u8],
) -> Result<TokenBridgeGovPayload<'_>> {
    let vaa = ZeroCopyEncodedVaa::parse(vaa_acc_data)?;
    require!(
        vaa.emitter_chain()? == SOLANA_CHAIN
            && vaa.emitter_address()? == crate::constants::GOVERNANCE_EMITTER,
        TokenBridgeError::InvalidGovernanceEmitter
    );

    // Because emitter_chain and emitter_address getters have succeeded, we can safely unwrap this
    // payload call.
    parse_gov_payload(vaa.payload().unwrap())
}

pub(super) fn parse_acc_data(vaa_acc_data: &[u8]) -> Result<TokenBridgeGovPayload<'_>> {
    let vaa = ZeroCopyEncodedVaa::parse(vaa_acc_data)?;
    parse_gov_payload(vaa.payload()?)
}

pub(super) fn parse_gov_payload(payload: Payload<'_>) -> Result<TokenBridgeGovPayload<'_>> {
    TokenBridgeGovPayload::try_from(payload)
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))
}
