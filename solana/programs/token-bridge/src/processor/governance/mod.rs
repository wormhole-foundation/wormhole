mod register_chain;
pub use register_chain::*;

mod secure_registered_emitter;
pub use secure_registered_emitter::*;

use crate::error::TokenBridgeError;
use anchor_lang::prelude::*;
use core_bridge_program::{constants::SOLANA_CHAIN, zero_copy::EncodedVaa};
use wormhole_raw_vaas::token_bridge::TokenBridgeGovPayload;

pub fn require_valid_governance_encoded_vaa(
    vaa_acc_data: &[u8],
) -> Result<TokenBridgeGovPayload<'_>> {
    let vaa_body = EncodedVaa::try_v1(vaa_acc_data).map(|vaa| vaa.body())?;
    require!(
        vaa_body.emitter_chain() == SOLANA_CHAIN
            && vaa_body.emitter_address() == crate::constants::GOVERNANCE_EMITTER,
        TokenBridgeError::InvalidGovernanceEmitter
    );

    // Because emitter_chain and emitter_address getters have succeeded, we can safely unwrap this
    // payload call.
    TokenBridgeGovPayload::try_from(vaa_body.payload())
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))
}
