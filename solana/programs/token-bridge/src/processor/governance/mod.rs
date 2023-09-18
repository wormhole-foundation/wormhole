mod register_chain;
pub use register_chain::*;

mod secure_registered_emitter;
pub use secure_registered_emitter::*;

use crate::error::TokenBridgeError;
use anchor_lang::prelude::*;
use core_bridge_program::sdk::{self as core_bridge_sdk, LoadZeroCopy};
use wormhole_raw_vaas::token_bridge::TokenBridgeGovPayload;

pub fn require_valid_governance_encoded_vaa(vaa_acc_info: &AccountInfo) -> Result<()> {
    let vaa = core_bridge_sdk::VaaAccount::load(vaa_acc_info)?;
    let (emitter_address, emitter_chain, _) = vaa.try_emitter_info()?;
    require!(
        emitter_chain == core_bridge_sdk::SOLANA_CHAIN
            && emitter_address == crate::constants::GOVERNANCE_EMITTER,
        TokenBridgeError::InvalidGovernanceEmitter
    );

    // Because emitter_chain and emitter_address getters have succeeded, we can safely unwrap this
    // payload call.
    TokenBridgeGovPayload::try_from(vaa.try_payload().unwrap())
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))?;

    // Done.
    Ok(())
}
