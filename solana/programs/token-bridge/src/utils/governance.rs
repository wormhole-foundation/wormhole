use crate::error::TokenBridgeError;
use anchor_lang::prelude::*;
use core_bridge_program::{constants::SOLANA_CHAIN, state::PostedVaaV1};
use wormhole_vaas::payloads::gov::{
    token_bridge::Decree, GovernanceMessage, GOVERNANCE_CHAIN, GOVERNANCE_EMITTER,
};

pub type PostedGovernanceVaaV1 = PostedVaaV1<GovernanceMessage<Decree>>;

pub(crate) fn require_valid_governance_posted_vaa<'ctx>(
    vaa: &'ctx Account<'_, PostedGovernanceVaaV1>,
) -> Result<&'ctx Decree> {
    require!(
        vaa.emitter_chain == GOVERNANCE_CHAIN && vaa.emitter_address == GOVERNANCE_EMITTER.0,
        TokenBridgeError::InvalidGovernanceEmitter
    );

    let decree = &vaa.payload.decree;

    // We need to check whether local governance VAAs are intended for Solana.
    let chain = match decree {
        Decree::RegisterChain(_) => SOLANA_CHAIN,
        Decree::ContractUpgrade(inner) => inner.chain,
        Decree::RecoverChainId(_) => return err!(TokenBridgeError::InvalidGovernanceAction),
    };
    require_eq!(
        chain,
        SOLANA_CHAIN,
        TokenBridgeError::GovernanceForAnotherChain
    );

    Ok(decree)
}
