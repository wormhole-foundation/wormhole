mod register_chain;
pub use register_chain::*;

use crate::error::TokenBridgeError;
use anchor_lang::prelude::*;
use core_bridge_program::{constants::SOLANA_CHAIN, zero_copy::EncodedVaa};
use wormhole_raw_vaas::token_bridge::TokenBridgeGovPayload;

pub fn require_valid_governance_encoded_vaa(
    vaa_acc_data: &[u8],
) -> Result<TokenBridgeGovPayload<'_>> {
    let vaa = EncodedVaa::try_v1(vaa_acc_data)?;
    require!(
        vaa.body().emitter_chain() == SOLANA_CHAIN
            && vaa.body().emitter_address() == crate::constants::GOVERNANCE_EMITTER,
        TokenBridgeError::InvalidGovernanceEmitter
    );

    // Because emitter_chain and emitter_address getters have succeeded, we can safely unwrap this
    // payload call.
    TokenBridgeGovPayload::try_from(vaa.body().payload())
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))
}
