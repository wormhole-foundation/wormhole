//! Parsers for Token bridge VAAs.
//!
//! Token bridging relies on VAA's that indicate custody/lockup/burn events in order to maintain
//! token parity between multiple chains. These parsers can be used to read these VAAs. It also
//! defines the Governance actions that this module supports, namely contract upgrades and chain
//! registrations.

use {
    crate::{
        require,
        vaa::GovHeader,
        Chain,
        WormholeError::{
            self,
            InvalidGovernanceChain,
        },
        VAA,
    },
    nom::{
        branch::alt,
        combinator::{
            flat_map,
            verify,
        },
        number::complete::u8,
    },
};

mod asset_meta;
mod contract_upgrade;
mod register_chain;
mod transfer;
mod transfer_with_payload;

pub use {
    asset_meta::*,
    contract_upgrade::*,
    register_chain::*,
    transfer::*,
    transfer_with_payload::*,
};


// Module: 000..TokenBridge in HEX.
const MODULE: [u8; 32] =
    hex_literal::hex!("000000000000000000000000000000000000000000546f6b656e427269646765");

// List of Actions (both standard and governance) the token bridge can process.
#[derive(Debug, PartialEq, Eq)]
pub enum Action {
    RegisterChain(RegisterChain),
    ContractUpgrade(ContractUpgrade),
    Transfer(Transfer),
    AssetMeta(AssetMeta),
    TransferWithPayload(TransferWithPayload),
}

// Implements the Action parser for token bridge actions.
impl crate::vaa::Action for Action {
    #[inline]
    fn from_vaa(vaa: &VAA, chain: Chain) -> Result<Self, WormholeError> {
        // Attempt to parse a GovHeader first as not all actions have one. On failure we know we
        // instead want to parse a non-governance action.
        let (i, header) = GovHeader::parse(&vaa.payload)?;

        // Parse Governance Actions.
        if header.module == MODULE {
            let valid_target = header.target == chain || header.target == Chain::Any;
            require!(valid_target, InvalidGovernanceChain);

            // Attempt to parse Governance actions.
            let (_, action) = match header.action {
                1 => RegisterChain::parse,
                2 => ContractUpgrade::parse,
                _ => return Err(WormholeError::UnknownGovernanceAction),
            }(i)?;

            Ok(action)
        }
        // Parse non-Governance Actions.
        else {
            let (_, action) = alt((
                flat_map(verify(u8, |&b| b == 1), |_| Transfer::parse),
                flat_map(verify(u8, |&b| b == 2), |_| AssetMeta::parse),
                flat_map(verify(u8, |&b| b == 3), |_| TransferWithPayload::parse),
            ))(&vaa.payload)?;

            Ok(action)
        }
    }
}
