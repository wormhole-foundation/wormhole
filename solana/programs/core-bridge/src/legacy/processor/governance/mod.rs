mod guardian_set_update;
pub use guardian_set_update::*;

mod set_message_fee;
pub use set_message_fee::*;

mod transfer_fees;
pub use transfer_fees::*;

mod upgrade_contract;
pub use upgrade_contract::*;
use wormhole_raw_vaas::core::CoreBridgeGovPayload;

use crate::{
    constants::SOLANA_CHAIN,
    error::CoreBridgeError,
    legacy::state::VaaAccountDetails,
    state::{Config, PartialPostedVaaV1},
};
use anchor_lang::prelude::*;

pub const GOVERNANCE_EMITTER: [u8; 32] = [
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4,
];

pub fn require_valid_governance_posted_vaa<'ctx>(
    vaa_details: VaaAccountDetails,
    vaa_acc_data: &'ctx [u8],
    guardian_set_index: u32,
    config: &'ctx Config,
) -> Result<CoreBridgeGovPayload<'ctx>> {
    let VaaAccountDetails {
        key: _,
        emitter_chain,
        emitter_address,
        sequence: _,
    } = vaa_details;

    // For the Core Bridge, we require that the current guardian set is used to sign this VAA.
    require_eq!(
        config.guardian_set_index,
        guardian_set_index,
        CoreBridgeError::LatestGuardianSetRequired
    );

    require!(
        emitter_chain == SOLANA_CHAIN && emitter_address == GOVERNANCE_EMITTER,
        CoreBridgeError::InvalidGovernanceEmitter
    );

    parse_gov_payload(vaa_acc_data)
}

pub fn parse_gov_payload(vaa_acc_data: &[u8]) -> Result<CoreBridgeGovPayload<'_>> {
    CoreBridgeGovPayload::parse(&vaa_acc_data[PartialPostedVaaV1::PAYLOAD_START..])
        .map_err(|_| CoreBridgeError::InvalidGovernanceVaa.into())
}
