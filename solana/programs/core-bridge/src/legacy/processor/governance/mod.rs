mod guardian_set_update;
pub use guardian_set_update::*;

mod set_message_fee;
pub use set_message_fee::*;

mod transfer_fees;
pub use transfer_fees::*;

mod upgrade_contract;
pub use upgrade_contract::*;

use crate::{
    constants::SOLANA_CHAIN,
    error::CoreBridgeError,
    state::{Config, PartialPostedVaaV1},
};
use anchor_lang::prelude::*;
use wormhole_io::Readable;

const GOVERNANCE_EMITTER: [u8; 32] = [
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4,
];

/// A.K.A. "Core" left padded with zeroes.
const GOVERNANCE_MODULE: [u8; 32] = *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00Core";

pub const GOVERNANCE_DECREE_START: usize = PartialPostedVaaV1::PAYLOAD_START + 32 + 1 + 2;

pub fn require_valid_governance_posted_vaa<'ctx>(
    vaa: &'ctx Account<'_, PartialPostedVaaV1>,
    config: &'ctx Config,
) -> Result<u8> {
    // For the Core Bridge, we require that the current guardian set is used to sign this VAA.
    require!(
        config.guardian_set_index == vaa.guardian_set_index,
        CoreBridgeError::LatestGuardianSetRequired
    );

    require!(
        vaa.emitter_chain == SOLANA_CHAIN && vaa.emitter_address == GOVERNANCE_EMITTER,
        CoreBridgeError::InvalidGovernanceEmitter
    );

    let acc_info: &AccountInfo = vaa.as_ref();
    let mut data: &[u8] = &acc_info.try_borrow_data()?;

    // Skip to governance message.
    data = &data[PartialPostedVaaV1::PAYLOAD_START..];

    // Encoded governance module must belong to this program.
    let module = <[u8; 32]>::read(&mut data)?;
    require!(
        module == GOVERNANCE_MODULE,
        CoreBridgeError::InvalidGovernanceAction
    );

    let action = u8::read(&mut data)?;

    // Either the target chain indicates a global governance action or it must be Solana.
    let target_chain = u16::read(&mut data)?;
    require!(
        target_chain == 0 || target_chain == SOLANA_CHAIN,
        CoreBridgeError::GovernanceForAnotherChain
    );

    // Return with encoded action for further validation.
    Ok(action)
}
