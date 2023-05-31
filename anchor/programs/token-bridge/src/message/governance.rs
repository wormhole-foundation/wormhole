use std::io;

use crate::error::TokenBridgeError;
use anchor_lang::prelude::*;
use core_bridge_program::{
    message::PostedGovernanceVaaV1,
    message::{GovernanceHeader, GovernanceMessage, TargetChain},
    WormDecode, WormEncode,
};

const GOVERNANCE_CHAIN: u16 = 1;
const GOVERNANCE_EMITTER: [u8; 32] = [
    0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
    0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4,
];

/// Governance module (A.K.A. "TokenBridge").
const GOVERNANCE_MODULE: [u8; 32] = [
    0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
    0x0, 0x0, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x42, 0x72, 0x69, 0x64, 0x67, 0x65,
];

#[derive(PartialEq, Eq)]
pub(crate) enum TokenBridgeGovernance {
    RegisterChain = 1,
    ContractUpgrade,
    /// NOTE: Unsupported on Solana.
    _RecoverChainId,
}

impl TryFrom<TokenBridgeGovernance> for GovernanceHeader {
    type Error = io::Error;

    fn try_from(action: TokenBridgeGovernance) -> std::result::Result<Self, Self::Error> {
        let (action, target) = match action {
            TokenBridgeGovernance::RegisterChain => (1, TargetChain::Global),
            TokenBridgeGovernance::ContractUpgrade => (2, TargetChain::Solana),
            _ => return Err(io::ErrorKind::InvalidData.into()),
        };

        Ok(GovernanceHeader {
            module: GOVERNANCE_MODULE.into(),
            action: action.into(),
            target,
        })
    }
}

pub(crate) fn require_valid_governance_posted_vaa<'ctx, D>(
    vaa: &'ctx Account<'_, PostedGovernanceVaaV1<D>>,
) -> Result<&'ctx GovernanceMessage<D>>
where
    D: Clone + WormDecode + WormEncode,
{
    require!(
        vaa.emitter_chain == GOVERNANCE_CHAIN.into()
            && vaa.emitter_address == GOVERNANCE_EMITTER.into(),
        TokenBridgeError::InvalidGovernanceEmitter
    );

    Ok(&vaa.payload)
}
