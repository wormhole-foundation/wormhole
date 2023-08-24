mod guardian_set_update;
pub use guardian_set_update::*;

mod set_message_fee;
pub use set_message_fee::*;

mod transfer_fees;
pub use transfer_fees::*;

mod upgrade_contract;
pub use upgrade_contract::*;
use wormhole_raw_vaas::core::{CoreBridgeDecree, CoreBridgeGovPayload};

use crate::{
    constants::SOLANA_CHAIN, error::CoreBridgeError, state::Config, zero_copy::PostedVaaV1,
};
use anchor_lang::prelude::*;

pub const GOVERNANCE_EMITTER: [u8; 32] = [
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4,
];

pub fn require_valid_posted_governance_vaa<'ctx>(
    vaa_acc_data: &'ctx [u8],
    config: &'ctx Config,
) -> Result<CoreBridgeDecree<'ctx>> {
    let vaa = PostedVaaV1::parse(vaa_acc_data)?;

    // For the Core Bridge, we require that the current guardian set is used to sign this VAA.
    require_eq!(
        config.guardian_set_index,
        vaa.guardian_set_index(),
        CoreBridgeError::LatestGuardianSetRequired
    );

    require!(
        vaa.emitter_chain() == SOLANA_CHAIN && vaa.emitter_address() == GOVERNANCE_EMITTER,
        CoreBridgeError::InvalidGovernanceEmitter
    );

    CoreBridgeGovPayload::parse(vaa.payload())
        .map(|msg| msg.decree())
        .map_err(|_| error!(CoreBridgeError::InvalidGovernanceVaa))
}
