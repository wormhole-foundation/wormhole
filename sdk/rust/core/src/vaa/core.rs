//! Parsers for core bridge VAAs.
//!
//! The main job of the bridge is to forward VAA's to other chains, however governance actions are
//! themselves VAAs and as such the bridge requires parsing Bridge specific VAAs. The core bridge
//! does not define any general VAA's, thus all the payloads in this file are expected to require
//! governance to be executed.

use crate::{
    require,
    Chain,
    GovHeader,
    WormholeError::{
        self,
        InvalidGovernanceChain,
        InvalidGovernanceModule,
    },
    VAA,
};

mod contract_upgrade;
mod guardian_set_change;
mod set_message_fee;
mod transfer_fees;

pub use {
    contract_upgrade::*,
    guardian_set_change::*,
    set_message_fee::*,
    transfer_fees::*,
};

// Module: 000..Core in HEX.
pub const MODULE: [u8; 32] =
    hex_literal::hex!("00000000000000000000000000000000000000000000000000000000436f7265");

/// Action in core represents a governance action targeted at the wormhole bridge itself.
#[derive(Debug, PartialEq, Eq)]
pub enum Action {
    ContractUpgrade(ContractUpgrade),
    GuardianSetChange(GuardianSetChange),
    SetMessageFee(SetMessageFee),
    TransferFees(TransferFees),
}

impl crate::vaa::Action for Action {
    #[inline]
    fn from_vaa(vaa: &VAA, chain: Chain) -> Result<Self, WormholeError> {
        // Parse GovHeader, which is always present in the Core contract.
        let (i, header) = GovHeader::parse(&vaa.payload)?;

        // Verify the `GovHeader` is valid.
        let valid_target = header.target == chain || header.target == Chain::Any;
        let valid_module = header.module == MODULE;
        require!(valid_target, InvalidGovernanceChain);
        require!(valid_module, InvalidGovernanceModule);

        // Parse the Payload.
        let (_, action) = match header.action {
            1 => ContractUpgrade::parse,
            2 => GuardianSetChange::parse,
            3 => SetMessageFee::parse,
            4 => TransferFees::parse,
            _ => return Err(WormholeError::UnknownGovernanceAction),
        }(i, header)?;

        Ok(action)
    }
}
