use anchor_lang::prelude::*;
use core_bridge_program::{
    constants::SOLANA_CHAIN, error::CoreBridgeError, state::PartialPostedVaaV1,
};
use wormhole_io::Readable;

const GOVERNANCE_CHAIN: u16 = 1;
const GOVERNANCE_EMITTER: [u8; 32] = [
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4,
];

/// A.K.A. "TokenBridge" left padded with zeroes.
const GOVERNANCE_MODULE: [u8; 32] = *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00TokenBridge";

pub(crate) const GOVERNANCE_DECREE_START: usize = PartialPostedVaaV1::PAYLOAD_START + 32 + 1 + 2;

pub(crate) fn require_valid_governance_posted_vaa(
    vaa: &Account<'_, PartialPostedVaaV1>,
) -> Result<u8> {
    crate::utils::require_valid_posted_vaa_key(&vaa.key())?;

    require!(
        vaa.emitter_chain == GOVERNANCE_CHAIN && vaa.emitter_address == GOVERNANCE_EMITTER,
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
