use crate::{
    constants::SOLANA_CHAIN,
    error::CoreBridgeError,
    state::{BridgeProgramData, PostedVaaV1},
};
use anchor_lang::prelude::*;
use wormhole_vaas::payloads::gov::{
    core_bridge::Decree, GovernanceMessage, GOVERNANCE_CHAIN, GOVERNANCE_EMITTER,
};

pub type PostedGovernanceVaaV1 = PostedVaaV1<GovernanceMessage<Decree>>;

pub fn require_valid_governance_posted_vaa<'ctx>(
    vaa: &'ctx Account<'_, PostedGovernanceVaaV1>,
    bridge: &'ctx BridgeProgramData,
) -> Result<&'ctx Decree> {
    // For the Core Bridge, we require that the current guardian set is used to sign this VAA.
    require!(
        bridge.guardian_set_index == vaa.guardian_set_index,
        CoreBridgeError::LatestGuardianSetRequired
    );

    require!(
        vaa.emitter_chain == GOVERNANCE_CHAIN && vaa.emitter_address == GOVERNANCE_EMITTER.0,
        CoreBridgeError::InvalidGovernanceEmitter
    );

    let decree = &vaa.payload.decree;

    // We need to check whether local governance VAAs are intended for Solana.
    let chain = match decree {
        Decree::ContractUpgrade(inner) => inner.chain,
        Decree::GuardianSetUpdate(_) => SOLANA_CHAIN,
        Decree::SetMessageFee(inner) => inner.chain,
        Decree::TransferFees(inner) => inner.chain,
        Decree::RecoverChainId(_) => return err!(CoreBridgeError::InvalidGovernanceAction),
    };
    require_eq!(
        chain,
        SOLANA_CHAIN,
        CoreBridgeError::GovernanceForAnotherChain
    );

    Ok(decree)
}
