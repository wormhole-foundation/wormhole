//! Parsers for NFT bridge VAAs.
//!
//! NFT bridging relies on VAA's that indicate custody/lockup/burn events in order to maintain
//! token parity between multiple chains. Parsers are provided here that can be used to read and
//! verify these events.

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
        combinator::{
            flat_map,
            verify,
        },
        number::complete::u8,
    },
};

mod contract_upgrade;
mod register_chain;
mod transfer;

pub use {
    contract_upgrade::*,
    register_chain::*,
    transfer::*,
};

const MODULE: [u8; 32] =
    hex_literal::hex!("00000000000000000000000000000000000000000000004e4654427269646765");

#[derive(Debug, PartialEq, Eq)]
pub enum Action {
    RegisterChain(RegisterChain),
    ContractUpgrade(ContractUpgrade),
    Transfer(Transfer),
}

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
            let (_, action) = flat_map(verify(u8, |&b| b == 1), |_| Transfer::parse)(&vaa.payload)?;
            Ok(action)
        }
    }
}
